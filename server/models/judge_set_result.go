package models

import (
	"time"
)

type JudgeSetResult struct {
	ID           uint            `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	SubmissionID uint            `gorm:"not null" json:"-"`
	CaseSet      CaseSet         `json:"-"`
	CaseSetID    uint            `gorm:"not null" json:"caseSetID"`
	Point        int             `json:"point"`
	Status       JudgementStatus `gorm:"not null; default:'0'" json:"status"`
	JudgeResults []JudgeResult   `json:"judgeResults"`
	ExecTime     time.Duration   `json:"execTime"`
	MemoryUsage  int64           `json:"memoryUsage"`
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

func (r *JudgeSetResult) FetchJudgeResults(sorted bool) {
	query := db
	if sorted {
		query = query.Order("id ASC")
	}
	query.Model(r).Related(&r.JudgeResults)
}

func (r *JudgeSetResult) GetJudgeResultsSorted() []JudgeResult {
	results := make([]JudgeResult, 0)
	db.Order("id ASC").Model(r).Related(&results)
	return results
}

func (r *JudgeSetResult) Delete() {
	r.FetchJudgeResults(false)
	for _, res := range r.JudgeResults {
		res.Delete()
	}
	db.Delete(JudgeSetResult{}, "id = ?", r.ID)
}
