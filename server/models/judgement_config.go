package models

import "time"

type JudgementConfig struct {
	ID              uint      `gorm:"primary_key" json:"id"`
	CreatedAt       time.Time `json:"-"`
	UpdatedAt       time.Time `json:"-"`
	JudgeSourceCode *string   `gorm:"type:text" json:"judgeSourceCode,omitempty"`
	LanguageID      *uint     `json:"languageID,omitempty"`
	Language        *Language `json:"language,omitempty"`
	Difference      float64   `json:"difference"`
}

func (d *JudgementConfig) FetchLanguage() {
	if d.LanguageID == nil {
		return
	}
	d.Language = &Language{}
	db.Model(d).Related(d.Language)
}
