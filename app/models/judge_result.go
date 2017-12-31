package models

import "time"

type JudgeResult struct {
	ID               uint            `gorm:"primary_key"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	JudgeSetResultID uint            `gorm:"not null"`
	TestCaseID       uint            `gorm:"not null"`
	JudgementStatus  JudgementStatus `gorm:"not null; default:'0'"`
}
