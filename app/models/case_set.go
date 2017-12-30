package models

import "time"

type CaseSet struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	ProblemID uint `gorm:"not null"`
	Point     int  `gorm:"not null; default:'0'"`
}
