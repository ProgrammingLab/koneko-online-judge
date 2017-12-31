package models

import (
	"time"
	"github.com/pkg/errors"
)

type Submission struct {
	ID          uint    `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UserID      uint    `gorm:"not null"`
	User        User
	LanguageID  uint    `gorm:"not null"`
	Language    Language
	SourceCode  string  `gorm:"type:text; not null"`
	Point       int
	Status      int     `gorm:"default:'0'"`
	ErrorLog    string  `gorm:"type:text"`
	ExecTime    float32 `gorm:"type:float"`
	MemoryUsage uint
	CodeBytes   uint
}

type SubmissionStatus int

const (
	InQueue           SubmissionStatus = 0
	Judging           SubmissionStatus = 1
	Accepted          SubmissionStatus = 2
	PresentationError SubmissionStatus = 3
	WrongAnswer       SubmissionStatus = 4
	TimeLimitExceeded SubmissionStatus = 5
	RuntimeError      SubmissionStatus = 6
	CompileError      SubmissionStatus = 7
	UnknownError      SubmissionStatus = 8
)

func Submit(submission *Submission) error {
	submission.CodeBytes = uint(len(submission.SourceCode))
	submission.ID = 0
	db.Create(submission)

	if submission.ID == 0 {
		return errors.New("something wrong")
	}

	return nil
}
