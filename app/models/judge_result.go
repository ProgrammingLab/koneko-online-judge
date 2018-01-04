package models

import "time"

type JudgeResult struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	JudgeSetResultID uint `gorm:"not null"`
	TestCase         TestCase
	TestCaseID       uint            `gorm:"not null"`
	Status           JudgementStatus `gorm:"not null; default:'0'"`
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
