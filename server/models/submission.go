package models

import (
	"time"

	"github.com/pkg/errors"
)

type Submission struct {
	ID              uint             `gorm:"primary_key" json:"id"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	UserID          uint             `gorm:"not null" json:"userID"`
	User            User             `json:"user"`
	ProblemID       uint             `gorm:"not null" json:"problemID"`
	Problem         Problem          `json:"problem"`
	LanguageID      uint             `gorm:"not null" json:"-"`
	Language        Language         `json:"language"`
	SourceCode      string           `gorm:"type:text; not null" json:"sourceCode"`
	Point           int              `json:"point"`
	Status          JudgementStatus  `gorm:"default:'0'" json:"status"`
	ErrorLog        string           `gorm:"type:text" json:"errorLog"`
	ExecTime        time.Duration    `json:"execTime"`
	MemoryUsage     int64            `json:"memoryUsage"`
	CodeBytes       uint             `json:"codeBytes"`
	JudgeSetResults []JudgeSetResult `json:"judgeSetResults"`
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
	notFound := db.Where(submission).First(submission).RecordNotFound()
	if notFound {
		return nil
	}
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

func (s *Submission) FetchJudgeSetResults(sorted bool) {
	query := db
	if sorted {
		query = query.Order("id ASC")
	}
	query.Model(s).Related(&s.JudgeSetResults)
}

func (s *Submission) FetchJudgeSetResultsDeeply(sorted bool) {
	query := db
	if sorted {
		query = query.Order("id ASC")
	}
	query.Model(s).Related(&s.JudgeSetResults)
	for i := range s.JudgeSetResults {
		s.JudgeSetResults[i].FetchJudgeResults(sorted)
	}
}

func (s *Submission) Delete() {
	s.FetchJudgeSetResults(false)
	for _, r := range s.JudgeSetResults {
		r.Delete()
	}

	db.Delete(s)
}
