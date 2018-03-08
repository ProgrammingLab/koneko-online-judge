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
	SubmissionID uint
}

const (
	imageNamePrefix    = "koneko-online-judge-image-"
	compileTimeLimit   = 5 * time.Second
	compileMemoryLimit = 256 * 1024 * 1024
)

func judge(submissionID uint) {
	jobs.Now(judgementJob{
		SubmissionID: submissionID,
	})
}

func (j judgementJob) Run() {
	submission := GetSubmission(j.SubmissionID)
	submission.Status = Judging
	db.Model(submission).Update("status", submission.Status)
	submission.FetchLanguage()
	submission.FetchProblem()
	submission.FetchJudgeSetResults(false)
	var (
		execTime    time.Duration
		memoryUsage int64
		point       int
		finalStatus = Accepted
	)

	compileWorker, compileRes := compile(submission.SourceCode[:], &submission.Language)
	if compileWorker == nil || compileRes == nil {
		finalStatus = UnknownError
		markAs(submission.JudgeSetResults, finalStatus)
	} else {
		defer compileWorker.Remove()
		logger.AppLog.Debugf("%v %v", compileRes.Status, compileRes.Stderr)

		if compileRes.Status != workers.StatusFinished {
			finalStatus = CompileError
			markAs(submission.JudgeSetResults, finalStatus)
			logger.AppLog.Debugf("compile error: worker status %v", compileRes.Status, compileRes.Stderr)
		} else {
			for _, r := range submission.JudgeSetResults {
				status, t, m := judgeCaseSet(&r, submission, compileWorker)
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

	submission.Point = point
	submission.Status = finalStatus
	submission.ExecTime = execTime
	submission.MemoryUsage = memoryUsage
	query := map[string]interface{}{
		"point":        point,
		"status":       finalStatus,
		"exec_time":    execTime,
		"memory_usage": memoryUsage,
	}
	db.Model(&Submission{ID: submission.ID}).Updates(query)

	if submission.Problem.ContestID != nil {
		p := &submission.Problem
		p.FetchContest()
		writer, err := p.Contest.IsWriter(submission.UserID)
		if err != nil {
			logger.AppLog.Errorf("error %+v", err)
			return
		}
		if p.Contest.IsOpen(submission.CreatedAt) && !writer {
			updateScore(submission, *submission.Problem.ContestID)
		}
	}
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

func judgeCaseSet(result *JudgeSetResult, submission *Submission, compileWorker *workers.Worker) (JudgementStatus, time.Duration, int64) {
	result.FetchCaseSet()
	result.FetchJudgeResults(false)

	setStatus := Accepted
	var (
		execTime    time.Duration
		memoryUsage int64
	)
	specialPoint := 0
	for _, r := range result.JudgeResults {
		specialPoint += judgeTestCase(&r, submission, compileWorker)
		execTime = MaxDuration(execTime, r.ExecTime)
		memoryUsage = MaxLong(memoryUsage, r.MemoryUsage)
		if r.Status != Accepted {
			setStatus = r.Status
		}
	}

	if setStatus == Accepted {
		switch submission.Problem.JudgeType {
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

func judgeTestCase(result *JudgeResult, submission *Submission, compileWorker *workers.Worker) int {
	result.Status = Judging
	db.Model(result).Update("status", result.Status)
	result.FetchTestCase()
	testCase := &result.TestCase

	res := execSubmission(submission, testCase, compileWorker)
	var point int
	result.Status, point = judgeExecResult(res, submission, testCase)
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

func execSubmission(submission *Submission, testCase *TestCase, compiled *workers.Worker) *workers.ExecResult {
	problem := &submission.Problem
	language := &submission.Language
	cmd := language.GetExecCommandSlice()
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, problem.TimeLimit, int64(problem.MemoryLimit*1024*1024), cmd)
	if err != nil {
		logger.AppLog.Errorf("exec: container create error %+v", err)
		return nil
	}
	defer w.Remove()

	err = compiled.CopyTo(language.ExeFileName, w)
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

func judgeExecResult(res *workers.ExecResult, submission *Submission, testCase *TestCase) (JudgementStatus, int) {
	problem := &submission.Problem

	switch problem.JudgeType {
	case JudgeTypeNormal:
		return judgeExecResultNormal(res, testCase), 0
	case JudgeTypePrecision:
		logger.AppLog.Errorf("'JudgeTypePrecision' is not implemented")
		return UnknownError, 0
	case JudgeTypeSpecial:
		return judgeExecResultSpecial(res, submission, testCase, problem.JudgementConfig)
	default:
		logger.AppLog.Errorf("judge type %v is not implemented", problem.JudgeType)
		return UnknownError, 0
	}
}

func judgeExecResultNormal(res *workers.ExecResult, testCase *TestCase) JudgementStatus {
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

func judgeExecResultSpecial(res *workers.ExecResult, submission *Submission, testCase *TestCase, config *JudgementConfig) (JudgementStatus, int) {
	compiled, compileRes := compile(*config.JudgeSourceCode, config.Language)
	if compiled == nil || compileRes == nil {
		return UnknownError, 0
	}
	defer compiled.Remove()
	if compileRes.Status != workers.StatusFinished {
		return CompileError, 0
	}

	l := submission.Language
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
	w.CopyContentToContainer([]byte(submission.SourceCode), l.FileName)

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
