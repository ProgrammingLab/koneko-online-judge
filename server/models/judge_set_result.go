package models

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type JudgeSetResult struct {
	ID           uint            `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	SubmissionID uint            `gorm:"not null" json:"-"`
	CaseSet      CaseSet         `json:"caseSet"`
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
	db.Delete(JudgeResult{}, "judge_set_result_id = ?", r.ID)
	db.Delete(JudgeSetResult{}, "id = ?", r.ID)
}

func (r *JudgeSetResult) setJudgementStatus(status JudgementStatus) error {
	r.Status = status
	err := db.Model(JudgeSetResult{}).Where("id = ?", r.ID).Update("status", r.Status).Error
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return err
	}

	r.FetchJudgeResults(false)
	for i := range r.JudgeResults {
		err := r.JudgeResults[i].setJudgementStatus(r.Status)
		if err != nil {
			logger.AppLog.Errorf("error: %+v", err)
			return err
		}
	}

	return nil
}
