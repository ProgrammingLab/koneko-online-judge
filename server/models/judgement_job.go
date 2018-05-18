package models

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/modules/workers"
	"github.com/gocraft/work"
)

type judgementJob struct {
	submissionID uint
	submission   *Submission

	compiled *workers.Worker
}

const (
	imageNamePrefix    = "koneko-online-judge-image-"
	compileTimeLimit   = 5 * time.Second
	compileMemoryLimit = 256 * 1024 * 1024
)

func judge(submissionID uint) error {
	_, err := enqueuer.Enqueue(judgementJobName, work.Q{submissionJobArgKey: submissionID})
	if err != nil {
		logger.AppLog.Errorf("job error: %+v", err)
	}
	return err
}

func compile(sourceCode string, language *Language) (*workers.Worker, *workers.ExecResult) {
	cmd := language.GetCompileCommandSlice()
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, compileTimeLimit, compileMemoryLimit, cmd)
	if err != nil {
		logger.AppLog.Errorf("compile: container create error %+v", err)
		return nil, nil
	}

	err = w.CopyContentToContainer([]byte(sourceCode), language.FileName)
	if err != nil {
		logger.AppLog.Errorf("compile: docker cp %+v", err)
		return nil, nil
	}

	res, err := w.Run("")
	if err != nil {
		logger.AppLog.Errorf("compile: container attach error %+v", err)
		return nil, nil
	}

	return w, res
}

func (j *judgementJob) Run() {
	defer j.Close()
	j.submission = GetSubmission(j.submissionID)
	if j.submission == nil {
		logger.AppLog.Infof("submission(id = %v) is deleted", j.submissionID)
		return
	}
	j.submission.SetStatus(StatusJudging)
	j.submission.FetchLanguage()
	j.submission.FetchProblem()
	j.submission.Problem.FetchJudgementConfig()
	j.submission.FetchJudgeSetResults(false)
	var (
		execTime    time.Duration
		memoryUsage int64
		point       = 0
		finalStatus = StatusUnknownError
	)

	defer func() {
		query := map[string]interface{}{
			"point":        point,
			"status":       finalStatus,
			"exec_time":    execTime,
			"memory_usage": memoryUsage,
		}
		db.Model(&Submission{ID: j.submission.ID}).Updates(query)
		onUpdateJudgementStatuses(j.submission.Problem.ContestID, *j.submission)
	}()

	var eval evaluator
	switch j.submission.Problem.JudgeType {
	case JudgeTypeNormal:
		eval = newSimpleEvaluator()
	case JudgeTypePrecision:
		eval = newPrecisionEvaluator(j.submission.Problem.JudgementConfig)
	case JudgeTypeSpecial:
		var err error
		eval, err = newSpecialEvaluator(j.submission.Problem.JudgementConfig, j.submission)
		if err != nil {
			logger.AppLog.Errorf("judge source code compile error: %+v", err)
			finalStatus = StatusUnknownError
			markAs(j.submission.JudgeSetResults, finalStatus)
			return
		}
	default:
		logger.AppLog.Errorf("%v is not implemented", j.submission.Problem.JudgeType)
		finalStatus = StatusUnknownError
		markAs(j.submission.JudgeSetResults, finalStatus)
		return
	}

	defer eval.remove()

	var compileRes *workers.ExecResult
	j.compiled, compileRes = compile(j.submission.SourceCode[:], &j.submission.Language)
	if j.compiled == nil || compileRes == nil {
		finalStatus = StatusUnknownError
		markAs(j.submission.JudgeSetResults, finalStatus)
	} else {
		logger.AppLog.Debugf("%v %v", compileRes.Status, compileRes.Stderr)

		if compileRes.Status != workers.StatusFinished {
			finalStatus = StatusCompileError
			markAs(j.submission.JudgeSetResults, finalStatus)
			logger.AppLog.Debugf("compile error: worker status %v", compileRes.Status, compileRes.Stderr)
		} else {
			for _, r := range j.submission.JudgeSetResults {
				r.FetchCaseSet()
				setEval := eval.next(&r.CaseSet, nil)
				j.judgeCaseSet(setEval, &r)
				execTime = MaxDuration(execTime, r.ExecTime)
				memoryUsage = MaxLong(memoryUsage, r.MemoryUsage)
			}
		}
	}

	if finalStatus != StatusCompileError {
		finalStatus, point = eval.evaluate()
	}

	j.submission.Point = point
	j.submission.Status = finalStatus
	j.submission.ExecTime = execTime
	j.submission.MemoryUsage = memoryUsage

	if j.submission.Problem.ContestID != nil {
		p := &j.submission.Problem
		p.FetchContest()
		writer, err := p.Contest.IsWriter(j.submission.UserID)
		if err != nil {
			logger.AppLog.Errorf("error %+v", err)
			return
		}
		if p.Contest.IsOpen(j.submission.CreatedAt) && !writer {
			updateScore(j.submission, *j.submission.Problem.ContestID)
		}
	}
}

func (j *judgementJob) Close() {
	if j.compiled == nil {
		return
	}
	j.compiled.Remove()
}

func markAs(setResults []JudgeSetResult, status JudgementStatus) {
	for _, s := range setResults {
		s.FetchJudgeResults(false)
		db.Model(s).Update("status", status)
		for _, r := range s.JudgeResults {
			db.Model(r).Update("status", status)
		}
	}
}

func (j *judgementJob) judgeCaseSet(evaluator caseSetEvaluator, result *JudgeSetResult) {
	result.FetchCaseSet()
	result.FetchJudgeResults(false)

	var (
		execTime    time.Duration
		memoryUsage int64
	)
	for _, r := range result.JudgeResults {
		j.judgeTestCase(evaluator, &r)
		execTime = MaxDuration(execTime, r.ExecTime)
		memoryUsage = MaxLong(memoryUsage, r.MemoryUsage)
	}

	result.Status, result.Point = evaluator.evaluate()
	result.ExecTime = execTime
	result.MemoryUsage = memoryUsage

	query := map[string]interface{}{
		"point":        result.Point,
		"status":       result.Status,
		"exec_time":    result.ExecTime,
		"memory_usage": result.MemoryUsage,
	}
	db.Model(&JudgeSetResult{ID: result.ID}).Updates(query)
}

func (j *judgementJob) judgeTestCase(evaluator caseSetEvaluator, result *JudgeResult) int {
	result.Status = StatusJudging
	db.Model(result).Update("status", result.Status)
	result.FetchTestCase()
	testCase := &result.TestCase

	res := j.execSubmission(testCase)
	var point int
	result.Status, point = evaluator.next(res, testCase)
	result.ExecTime = res.ExecTime
	result.MemoryUsage = res.MemoryUsage / 1024

	query := map[string]interface{}{
		"status":       result.Status,
		"exec_time":    result.ExecTime,
		"memory_usage": result.MemoryUsage,
	}
	db.Model(&JudgeResult{ID: result.ID}).Updates(query)
	return point
}

func (j *judgementJob) execSubmission(testCase *TestCase) *workers.ExecResult {
	problem := &j.submission.Problem
	language := &j.submission.Language
	cmd := language.GetExecCommandSlice()
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, problem.TimeLimit, int64(problem.MemoryLimit*1024*1024), cmd)
	if err != nil {
		logger.AppLog.Errorf("exec: container create error %+v", err)
		return nil
	}
	defer w.Remove()

	err = j.compiled.CopyTo(language.ExeFileName, w)
	if err != nil {
		logger.AppLog.Errorf("exec: docker cp error %+v", err)
		return nil
	}

	res, err := w.Run(testCase.Input[:])
	if err != nil {
		logger.AppLog.Errorf("exec: container attach error %+v", err)
	}
	return res
}
