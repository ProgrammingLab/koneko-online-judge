package models

import "time"

type JudgeSetResult struct {
	ID           uint `gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SubmissionID uint `gorm:"not null"`
	CaseSetID    uint `gorm:"not null"`
	Point        int
}
