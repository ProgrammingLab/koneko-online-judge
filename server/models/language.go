package models

import "time"

type Language struct {
	ID             uint      `gorm:"primary_key" json:"-"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
	ImageName      string    `gorm:"not null" json:"-"`
	DisplayName    string    `gorm:"not null; unique_index" json:"displayName"`
	FileName       string    `gorm:"not null" json:"-"`
	ExeFileName    string    `gorm:"not null" json:"-"`
	CompileCommand string    `gorm:"not null" json:"compileCommand"`
	ExecCommand    string    `gorm:"not null" json:"execCommand"`
}

func GetAllLanguages() []*Language {
	result := make([]*Language, 0)
	db.Find(&result)
	return result
}

func GetLanguageByDisplayName(displayName string) *Language {
	result := &Language{DisplayName: displayName}
	notFound := db.Where(result).First(result).RecordNotFound()
	if notFound {
		return nil
	}
	return result
}
