package models

import (
	"strings"
	"time"
)

type Language struct {
	ID             uint      `gorm:"primary_key" json:"id"`
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

func GetLanguage(id uint) *Language {
	res := &Language{}
	notFound := db.Where("id = ?", id).First(res).RecordNotFound()
	if notFound {
		return nil
	}
	return res
}

func (l Language) GetCompileCommandSlice() []string {
	return strings.Split(l.CompileCommand, " ")
}

func (l Language) GetExecCommandSlice() []string {
	return strings.Split(l.ExecCommand, " ")
}
