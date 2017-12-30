package models

import "time"

type TestCase struct {
	ID        uint   `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	CaseSetID uint   `gorm:"not null"`
	Input     string `gorm:"type:longtext; not null"`
	Output    string `gorm:"type:longtext; not null"`
}
