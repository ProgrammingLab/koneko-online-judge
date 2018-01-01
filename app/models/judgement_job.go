package models

import (
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/gedorinku/koneko-online-judge/app/models/docker"
	"github.com/revel/revel"
)

type judgementJob struct {
	SubmissionID uint
}

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
	revel.AppLog.Infof("job %v %v", point, finalStatus)
	db.Save(submission)
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
	revel.AppLog.Infof("update %v", result.Status)
	db.Save(result)

	return setStatus
}

func judgeTestCase(result *JudgeResult, submission *Submission) JudgementStatus {
	defer db.Model(result).Updates(JudgeResult{Status: result.Status})

	result.FetchTestCase()
	language := &submission.Language
	container := docker.CreateContainer(language.ImageName, submission.Problem.MemoryLimit, result.TestCase.Input)
	if container == nil {
		result.Status = UnknownError
		return result.Status
	}

	code, _ := container.Compile(language.CompileCommand)
	if code != 0 {
		result.Status = CompileError
		return result.Status
	}

	result.Status = Accepted
	return result.Status
}
