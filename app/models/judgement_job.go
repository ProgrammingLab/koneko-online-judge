package models

import (
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/gedorinku/koneko-online-judge/app/models/docker"
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
