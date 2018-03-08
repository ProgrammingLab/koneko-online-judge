package models

import "time"

type JudgeResult struct {
	ID               uint            `gorm:"primary_key" json:"id"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	JudgeSetResultID uint            `gorm:"not null" json:"-"`
	TestCase         TestCase        `json:"-"`
	TestCaseID       uint            `gorm:"not null" json:"-"`
	Status           JudgementStatus `gorm:"not null; default:'0'" json:"status"`
	ExecTime         time.Duration   `json:"execTime"`
	MemoryUsage      int64           `json:"memoryUsage"`
}

func newJudgeResult(testCase *TestCase, setResult *JudgeSetResult) {
	result := &JudgeResult{
		JudgeSetResultID: setResult.ID,
		TestCaseID:       testCase.ID,
	}
	db.Create(result)
}

func (r *JudgeResult) FetchTestCase() {
	db.Model(r).Related(&r.TestCase)
}

func (r *JudgeResult) Delete() {
	db.Delete(r)
}
