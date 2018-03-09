package models

import (
	"strconv"
	"strings"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models/workers"
	"github.com/gedorinku/koneko-online-judge/server/modules/jobs"
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

func judge(submissionID uint) {
	jobs.Now(&judgementJob{
		submissionID: submissionID,
		compiled:     nil,
	})
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
	j.submission.Status = Judging
	db.Model(j.submission).Update("status", j.submission.Status)
	j.submission.FetchLanguage()
	j.submission.FetchProblem()
	j.submission.Problem.FetchJudgementConfig()
	j.submission.FetchJudgeSetResults(false)
	var (
		execTime    time.Duration
		memoryUsage int64
		point       int
		finalStatus = Accepted
	)

	var compileRes *workers.ExecResult
	j.compiled, compileRes = compile(j.submission.SourceCode[:], &j.submission.Language)
	if j.compiled == nil || compileRes == nil {
		finalStatus = UnknownError
		markAs(j.submission.JudgeSetResults, finalStatus)
	} else {
		logger.AppLog.Debugf("%v %v", compileRes.Status, compileRes.Stderr)

		if compileRes.Status != workers.StatusFinished {
			finalStatus = CompileError
			markAs(j.submission.JudgeSetResults, finalStatus)
			logger.AppLog.Debugf("compile error: worker status %v", compileRes.Status, compileRes.Stderr)
		} else {
			for _, r := range j.submission.JudgeSetResults {
				status, t, m := j.judgeCaseSet(&r)
				execTime = MaxDuration(execTime, t)
				memoryUsage = MaxLong(memoryUsage, m)
				point += r.Point
				if status == Accepted {
					continue
				}
				finalStatus = status
			}
		}
	}

	j.submission.Point = point
	j.submission.Status = finalStatus
	j.submission.ExecTime = execTime
	j.submission.MemoryUsage = memoryUsage
	query := map[string]interface{}{
		"point":        point,
		"status":       finalStatus,
		"exec_time":    execTime,
		"memory_usage": memoryUsage,
	}
	db.Model(&Submission{ID: j.submission.ID}).Updates(query)

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

func (j *judgementJob) judgeCaseSet(result *JudgeSetResult) (JudgementStatus, time.Duration, int64) {
	result.FetchCaseSet()
	result.FetchJudgeResults(false)

	setStatus := Accepted
	var (
		execTime    time.Duration
		memoryUsage int64
	)
	specialPoint := 0
	for _, r := range result.JudgeResults {
		specialPoint += j.judgeTestCase(&r)
		execTime = MaxDuration(execTime, r.ExecTime)
		memoryUsage = MaxLong(memoryUsage, r.MemoryUsage)
		if r.Status != Accepted {
			setStatus = r.Status
		}
	}

	if setStatus == Accepted {
		switch j.submission.Problem.JudgeType {
		case JudgeTypeNormal:
			result.Point = result.CaseSet.Point
		case JudgeTypePrecision:
			logger.AppLog.Errorf("'JudgeTypePrecision' is not implemented")
			setStatus = UnknownError
		case JudgeTypeSpecial:
			result.Point = specialPoint
		}
	}

	result.Status = setStatus
	result.ExecTime = execTime
	result.MemoryUsage = memoryUsage
	query := map[string]interface{}{
		"point":        result.Point,
		"status":       setStatus,
		"exec_time":    execTime,
		"memory_usage": memoryUsage,
	}
	db.Model(&JudgeSetResult{ID: result.ID}).Updates(query)

	return setStatus, execTime, memoryUsage
}

func (j *judgementJob) judgeTestCase(result *JudgeResult) int {
	result.Status = Judging
	db.Model(result).Update("status", result.Status)
	result.FetchTestCase()
	testCase := &result.TestCase

	res := j.execSubmission(testCase)
	var point int
	result.Status, point = j.judgeExecResult(res, testCase)
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

func (j *judgementJob) judgeExecResult(res *workers.ExecResult, testCase *TestCase) (JudgementStatus, int) {
	problem := &j.submission.Problem

	switch problem.JudgeType {
	case JudgeTypeNormal:
		return j.judgeExecResultNormal(res, testCase), 0
	case JudgeTypePrecision:
		logger.AppLog.Errorf("'JudgeTypePrecision' is not implemented")
		return UnknownError, 0
	case JudgeTypeSpecial:
		return j.judgeExecResultSpecial(res, testCase, problem.JudgementConfig)
	default:
		logger.AppLog.Errorf("judge type %v is not implemented", problem.JudgeType)
		return UnknownError, 0
	}
}

func (j *judgementJob) judgeExecResultNormal(res *workers.ExecResult, testCase *TestCase) JudgementStatus {
	if res == nil {
		return UnknownError
	}

	switch res.Status {
	case workers.StatusMemoryLimitExceeded:
		return MemoryLimitExceeded
	case workers.StatusTimeLimitExceeded:
		return TimeLimitExceeded
	case workers.StatusRuntimeError:
		return RuntimeError
	case workers.StatusFinished:
		if res.Stdout == testCase.Output {
			return Accepted
		}
		if strings.TrimSpace(res.Stdout) == strings.TrimSpace(testCase.Output) {
			return PresentationError
		}
		return WrongAnswer
	default:
		return UnknownError
	}
}

func (j *judgementJob) judgeExecResultSpecial(res *workers.ExecResult, testCase *TestCase, config *JudgementConfig) (JudgementStatus, int) {
	compiled, compileRes := compile(*config.JudgeSourceCode, config.Language)
	if compiled == nil || compileRes == nil {
		return UnknownError, 0
	}
	defer compiled.Remove()
	if compileRes.Status != workers.StatusFinished {
		return CompileError, 0
	}

	l := j.submission.Language
	const (
		input      = "in"
		output     = "out"
		userOutput = "submission"
	)
	cmd := append(config.Language.GetExecCommandSlice(), input, output, userOutput, l.FileName)
	w, err := workers.NewWorker(imageNamePrefix+config.Language.ImageName, compileTimeLimit, compileMemoryLimit, cmd)
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return UnknownError, 0
	}
	defer w.Remove()

	w.CopyContentToContainer([]byte(testCase.Input), input)
	w.CopyContentToContainer([]byte(testCase.Output), output)
	w.CopyContentToContainer([]byte(res.Stdout), userOutput)
	w.CopyContentToContainer([]byte(j.submission.SourceCode), l.FileName)

	judged, err := w.Run("")
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return UnknownError, 0
	}

	point, _ := strconv.Atoi(judged.Stdout)
	if judged.Status == workers.StatusFinished {
		return Accepted, point
	}
	return WrongAnswer, 0
}
