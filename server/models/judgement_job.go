package models

import (
	"strings"

	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models/workers"
	"github.com/gedorinku/koneko-online-judge/server/modules/jobs"
)

type judgementJob struct {
	SubmissionID uint
}

const imageNamePrefix = "koneko-online-judge-image-"

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

	compileWorker, compileRes := compile(submission)
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
		if p.Contest.IsOpen(submission.CreatedAt) {
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
	for _, r := range result.JudgeResults {
		status, t, m := judgeTestCase(&r, submission, compileWorker)
		execTime = MaxDuration(execTime, t)
		memoryUsage = MaxLong(memoryUsage, m)
		if status != Accepted {
			setStatus = status
		}
	}

	if setStatus == Accepted {
		result.Point = result.CaseSet.Point
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

func judgeTestCase(result *JudgeResult, submission *Submission, compileWorker *workers.Worker) (JudgementStatus, time.Duration, int64) {
	result.Status = Judging
	db.Model(result).Update("status", result.Status)
	result.FetchTestCase()
	testCase := &result.TestCase

	res := execSubmission(submission, testCase, compileWorker)
	result.Status = toJudgementStatus(res, testCase)
	result.ExecTime = res.ExecTime
	result.MemoryUsage = res.MemoryUsage / 1024

	query := map[string]interface{}{
		"status":       result.Status,
		"exec_time":    result.ExecTime,
		"memory_usage": result.MemoryUsage,
	}
	db.Model(&JudgeResult{ID: result.ID}).Updates(query)
	return result.Status, result.ExecTime, result.MemoryUsage
}

func compile(submission *Submission) (*workers.Worker, *workers.ExecResult) {
	language := &submission.Language
	cmd := strings.Split(language.CompileCommand, " ")
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, 5*time.Second, int64(256*1024*1024), cmd)
	if err != nil {
		logger.AppLog.Errorf("compile: container create error %+v", err)
		return nil, nil
	}

	err = w.CopyContentToContainer([]byte(submission.SourceCode), language.FileName)
	if err != nil {
		logger.AppLog.Errorf("compile: docker cp %+v", err)
		return nil, nil
	}

	res, err := w.Run("")
	if err != nil {
		logger.AppLog.Errorf("compile: container attach error %+v", err)
	}

	return w, res
}

func execSubmission(submission *Submission, testCase *TestCase, compiled *workers.Worker) *workers.ExecResult {
	problem := &submission.Problem
	language := &submission.Language
	cmd := strings.Split(language.ExecCommand, " ")
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

func toJudgementStatus(res *workers.ExecResult, testCase *TestCase) JudgementStatus {
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
