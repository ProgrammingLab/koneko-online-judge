package models

import (
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/gedorinku/koneko-online-judge/app/models/workers"
	"github.com/revel/revel"
	"strings"
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
	submission.FetchLanguage()
	submission.FetchJudgeSetResults()

	point := 0
	finalStatus := Accepted
	for _, r := range submission.JudgeSetResults {
		status := judgeCaseSet(&r, submission)
		point += r.Point
		if status == Accepted {
			continue
		}
		finalStatus = status
	}

	submission.Point = point
	submission.Status = finalStatus
	db.Model(&Submission{ID: submission.ID}).Updates(map[string]interface{}{"point": point, "status": finalStatus})
}

func judgeCaseSet(result *JudgeSetResult, submission *Submission) JudgementStatus {
	result.FetchCaseSet()
	result.FetchJudgeResults()

	setStatus := Accepted
	for _, r := range result.JudgeResults {
		status := judgeTestCase(&r, submission)
		if status != Accepted {
			setStatus = status
		}
	}

	if setStatus == Accepted {
		result.Point = result.CaseSet.Point
	}

	result.Status = setStatus
	db.Model(&JudgeSetResult{ID: result.ID}).Updates(map[string]interface{}{"point": result.Point, "status": result.Status})

	return setStatus
}

func judgeTestCase(result *JudgeResult, submission *Submission) JudgementStatus {
	defer func() {
		db.Model(&JudgeResult{ID: result.ID}).Update("status", result.Status)
	}()

	result.FetchTestCase()
	testCase := &result.TestCase
	// TODO コンパイル結果のキャッシュ
	compileWorker, compileRes := compile(submission)
	if compileWorker == nil || compileRes == nil {
		result.Status = UnknownError
		return result.Status
	}
	if compileRes.Status != workers.StatusFinished {
		result.Status = CompileError
		return result.Status
	}
	defer compileWorker.Remove()

	res := execSubmission(submission, testCase, compileWorker)
	result.Status = toJudgementStatus(res, testCase)

	return result.Status
}

func compile(submission *Submission) (*workers.Worker, *workers.ExecResult) {
	problem := &submission.Problem
	language := &submission.Language
	cmd := strings.Split(language.CompileCommand, " ")
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, int64(problem.TimeLimit*1000), int64(problem.MemoryLimit*1000), cmd)
	if err != nil {
		revel.AppLog.Errorf("compile: container create error", err)
		return nil, nil
	}

	res, err := w.Run("")
	if err != nil {
		revel.AppLog.Errorf("compile: container attach error", err)
	}

	return w, res
}

func execSubmission(submission *Submission, testCase *TestCase, compiled *workers.Worker) *workers.ExecResult {
	problem := &submission.Problem
	language := &submission.Language
	cmd := strings.Split(language.CompileCommand, " ")
	w, err := workers.NewWorker(imageNamePrefix+language.ImageName, int64(problem.TimeLimit*1000), int64(problem.MemoryLimit*1000), cmd)
	if err != nil {
		revel.AppLog.Errorf("exec: container create error", err)
		return nil
	}

	path := "/tmp/" + language.ExeFileName
	err = compiled.CopyTo(path, path, w.ID)
	if err != nil {
		revel.AppLog.Errorf("exec: docker cp error", err)
		return nil
	}

	res, err := w.Run(testCase.Input[:])
	if err != nil {
		revel.AppLog.Errorf("exec: container attach error", err)
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
		return WrongAnswer
	default:
		return UnknownError
	}
}
