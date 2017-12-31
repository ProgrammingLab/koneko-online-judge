package models

import (
	"time"
	"github.com/pkg/errors"
	"github.com/gedorinku/koneko-online-judge/app/deamon"
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

type JudgementStatus int

const (
	InQueue           JudgementStatus = 0
	Judging           JudgementStatus = 1
	Accepted          JudgementStatus = 2
	PresentationError JudgementStatus = 3
	WrongAnswer       JudgementStatus = 4
	TimeLimitExceeded JudgementStatus = 5
	RuntimeError      JudgementStatus = 6
	CompileError      JudgementStatus = 7
	UnknownError      JudgementStatus = 8
)

func Submit(submission *Submission) error {
	submission.CodeBytes = uint(len(submission.SourceCode))
	submission.ID = 0
	db.Create(submission)

	if submission.ID == 0 {
		return errors.New("something wrong")
	}

	deamon.Judge(submission.ID)

	return nil
}

func GetSubmission(submissionID uint) *Submission {
	submission := &Submission{ID: submissionID,}
	db.Where(submission).First(submission)
	return submission
}
