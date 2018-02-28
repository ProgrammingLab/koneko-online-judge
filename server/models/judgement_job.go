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

	compileWorker, err := compile(submission)
	if err != nil {
		finalStatus = UnknownError
		markAs(submission.JudgeSetResults, finalStatus)
	} else {
		defer compileWorker.Remove()

		if compileWorker.Status != workers.StatusFinished {
			finalStatus = CompileError
			markAs(submission.JudgeSetResults, finalStatus)
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

	w, err := execSubmission(submission, testCase, compileWorker)
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
	}
	result.Status = toJudgementStatus(w, testCase)
	result.ExecTime = w.ExecTime
	result.MemoryUsage = w.MemoryUsage / 1024

	query := map[string]interface{}{
		"status":       result.Status,
		"exec_time":    result.ExecTime,
		"memory_usage": result.MemoryUsage,
	}
	db.Model(&JudgeResult{ID: result.ID}).Updates(query)
	return result.Status, result.ExecTime, result.MemoryUsage
}

func compile(submission *Submission) (*workers.Worker, error) {
	language := &submission.Language
	cmd := strings.Split(language.CompileCommand, " ")
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, 5*time.Second, int64(256*1024*1024), cmd)
	if err != nil {
		logger.AppLog.Errorf("compile: container create error %+v", err)
		return nil, err
	}

	err = w.CopyContentToContainer([]byte(submission.SourceCode), language.FileName)
	if err != nil {
		logger.AppLog.Errorf("compile: docker cp %+v", err)
		return nil, err
	}

	_, err = w.Output()
	if err != nil {
		logger.AppLog.Errorf("compile: container attach error %+v", err)
	}

	return w, nil
}

func execSubmission(submission *Submission, testCase *TestCase, compiled *workers.Worker) (*workers.Worker, error) {
	problem := &submission.Problem
	language := &submission.Language
	cmd := strings.Split(language.ExecCommand, " ")
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, problem.TimeLimit, int64(problem.MemoryLimit*1024*1024), cmd)
	if err != nil {
		logger.AppLog.Errorf("exec: container create error %+v", err)
		return nil, err
	}
	defer w.Remove()

	err = compiled.CopyTo(language.ExeFileName, w)
	if err != nil {
		logger.AppLog.Errorf("exec: docker cp error %+v", err)
		return nil, err
	}

	w.Stdin.Write([]byte(testCase.Input))
	if err := w.Start(); err != nil {
		logger.AppLog.Errorf("exec: container attach error %+v", err)
		return nil, err
	}
	if err := w.Wait(); err != nil {
		logger.AppLog.Errorf("exec: %+v", err)
		return nil, err
	}
	return w, nil
}

func toJudgementStatus(res *workers.Worker, testCase *TestCase) JudgementStatus {
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
		buf := make([]byte, 0, workers.OutputLimit)
		n, _ := res.Stdout.Read(buf)
		out := string(buf[:n])
		if out == testCase.Output {
			return Accepted
		}
		if strings.TrimSpace(out) == strings.TrimSpace(testCase.Output) {
			return PresentationError
		}
		return WrongAnswer
	default:
		return UnknownError
	}
}
