package models

import (
	"time"

	"github.com/pkg/errors"
)

type Submission struct {
	ID              uint `gorm:"primary_key"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	UserID          uint `gorm:"not null"`
	User            User
	ProblemID       uint `gorm:"not null"`
	Problem         Problem
	LanguageID      uint `gorm:"not null"`
	Language        Language
	SourceCode      string `gorm:"type:text; not null"`
	Point           int
	Status          JudgementStatus `gorm:"default:'0'"`
	ErrorLog        string          `gorm:"type:text"`
	ExecTime        float32         `gorm:"type:float"`
	MemoryUsage     uint
	CodeBytes       uint
	JudgeSetResults []JudgeSetResult
}

type JudgementStatus int

const (
	InQueue             JudgementStatus = 0
	Judging             JudgementStatus = 1
	Accepted            JudgementStatus = 2
	PresentationError   JudgementStatus = 3
	WrongAnswer         JudgementStatus = 4
	TimeLimitExceeded   JudgementStatus = 5
	MemoryLimitExceeded JudgementStatus = 6
	RuntimeError        JudgementStatus = 7
	CompileError        JudgementStatus = 8
	OutputLimitExceeded JudgementStatus = 9
	UnknownError        JudgementStatus = 10
)

func Submit(submission *Submission) error {
	submission.CodeBytes = uint(len(submission.SourceCode))
	submission.ID = 0
	db.Create(submission)

	if submission.ID == 0 {
		return errors.New("something wrong")
	}

	initJudgeSetResults(submission)
	judge(submission.ID)

	return nil
}

func GetSubmission(submissionID uint) *Submission {
	submission := &Submission{ID: submissionID}
	db.Where(submission).First(submission)
	return submission
}

func (s *Submission) FetchUser() {
	db.Model(s).Related(&s.User)
}

func (s *Submission) FetchLanguage() {
	db.Model(s).Related(&s.Language)
}

func (s *Submission) FetchProblem() {
	db.Model(s).Related(&s.Problem)
}

func (s *Submission) FetchJudgeSetResults() {
	db.Model(s).Related(&s.JudgeSetResults)
}
