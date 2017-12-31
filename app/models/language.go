package models

import "time"

type Language struct {
	ID             uint   `gorm:"primary_key"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Name           string `gorm:"not null;unique_index"`
	DisplayName    string `gorm:"not null"`
	FileName       string `gorm:"not null"`
	CompileCommand string `gorm:"not null"`
	ExecCommand    string `gorm:"not null"`
}
