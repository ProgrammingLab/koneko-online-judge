package models

import "time"

type Language struct {
	ID             uint   `gorm:"primary_key"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Name           string `gorm:"not null; unique_index"`
	DisplayName    string `gorm:"not null; unique_index"`
	FileName       string `gorm:"not null"`
	CompileCommand string `gorm:"not null"`
	ExecCommand    string `gorm:"not null"`
}

func GetAllLanguages() []*Language {
	result := make([]*Language, 0)
	db.Find(&result)
	return result
}

func GetLanguageByDisplayName(displayName string) *Language {
	result := &Language{DisplayName:displayName}
	db.Where(result).First(result)
	return result
}
