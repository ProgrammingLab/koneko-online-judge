package models

import (
	"time"
)

type JudgeSetResult struct {
	ID           uint `gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SubmissionID uint `gorm:"not null"`
	CaseSet      CaseSet
	CaseSetID    uint `gorm:"not null"`
	Point        int
	Status       JudgementStatus `gorm:"not null; default:'0'"`
	JudgeResults []JudgeResult
}

func initJudgeSetResults(submission *Submission) {
	submission.FetchProblem()
	problem := &submission.Problem
	problem.FetchCaseSets()

	for _, s := range problem.CaseSets {
		newJudgeSetResult(&s, submission)
	}
}

func newJudgeSetResult(set *CaseSet, submission *Submission) {
	result := &JudgeSetResult{
		SubmissionID: submission.ID,
		CaseSetID:    set.ID,
	}
	db.Create(result)

	set.FetchTestCases()
	for _, c := range set.TestCases {
		newJudgeResult(&c, result)
	}
}

func (r *JudgeSetResult) FetchCaseSet() {
	db.Model(r).Related(&r.CaseSet)
}

func (r *JudgeSetResult) FetchJudgeResults() {
	db.Model(r).Related(&r.JudgeResults)
}
