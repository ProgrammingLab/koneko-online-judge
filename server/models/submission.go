package models

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
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
	LanguageID      uint             `gorm:"not null" json:"languageID"`
	Language        Language         `json:"language"`
	SourceCode      string           `gorm:"type:text; not null" json:"sourceCode"`
	Point           int              `json:"point"`
	Status          JudgementStatus  `gorm:"default:'0'" json:"status"`
	ErrorLog        string           `gorm:"type:text" json:"errorLog"`
	ExecTime        time.Duration    `json:"execTime"`
	MemoryUsage     int64            `json:"memoryUsage"`
	CodeBytes       uint             `json:"codeBytes"`
	JudgeSetResults []JudgeSetResult `json:"judgeSetResults,omitempty"`
	ContestID       *uint            `json:"-"`
}

type JudgementStatus int

const (
	StatusInQueue             JudgementStatus = 0
	StatusJudging             JudgementStatus = 1
	StatusAccepted            JudgementStatus = 2
	StatusPresentationError   JudgementStatus = 3
	StatusWrongAnswer         JudgementStatus = 4
	StatusTimeLimitExceeded   JudgementStatus = 5
	StatusMemoryLimitExceeded JudgementStatus = 6
	StatusRuntimeError        JudgementStatus = 7
	StatusCompileError        JudgementStatus = 8
	StatusOutputLimitExceeded JudgementStatus = 9
	StatusUnknownError        JudgementStatus = 10
)

func Submit(submission *Submission) error {
	submission.CodeBytes = uint(len(submission.SourceCode))
	submission.ID = 0
	db.Create(submission)

	if submission.ID == 0 {
		return errors.New("something wrong")
	}

	submission.FetchProblem()
	onUpdateJudgementStatuses(submission.Problem.ContestID, *submission)
	initJudgeSetResults(submission)

	return judge(submission.ID)
}

func GetSubmission(submissionID uint) *Submission {
	submission := &Submission{ID: submissionID}
	notFound := db.Where(submission).First(submission).RecordNotFound()
	if notFound {
		return nil
	}
	return submission
}

func fetchSubmissionFieldsWithCache(out []Submission) error {
	users := make(map[uint]User)
	problems := make(map[uint]Problem)
	languages := make(map[uint]Language)

	for i := range out {
		if u, ok := users[out[i].UserID]; ok {
			out[i].User = u
		} else {
			user := User{}
			err := db.Model(User{}).Where("id = ?", out[i].UserID).Scan(&user).Error
			if err != nil {
				return err
			}
			user.Email = ""
			out[i].User = user
			users[user.ID] = user
		}

		if p, ok := problems[out[i].ProblemID]; ok {
			out[i].Problem = p
		} else {
			problem := Problem{}
			err := db.Model(Problem{}).Where("id = ?", out[i].ProblemID).Scan(&problem).Error
			if err != nil {
				return err
			}
			out[i].Problem = problem
			problems[problem.ID] = problem
		}

		if l, ok := languages[out[i].LanguageID]; ok {
			out[i].Language = l
		} else {
			language := Language{}
			err := db.Model(Language{}).Where("id = ?", out[i].LanguageID).Scan(&language).Error
			if err != nil {
				return err
			}
			out[i].Language = language
			languages[language.ID] = language
		}
	}

	return nil
}

func (s *Submission) IsWrong() bool {
	stat := s.Status
	return stat == StatusWrongAnswer || stat == StatusTimeLimitExceeded || stat == StatusMemoryLimitExceeded || stat == StatusRuntimeError
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

func (s *Submission) CanView(session *UserSession) bool {
	s.FetchProblem()
	if s.Problem.ContestID == nil || s.Problem.CanEdit(session) {
		return true
	}

	s.Problem.FetchContest()
	return s.Problem.CanView(session) && s.Problem.Contest.Started() && !s.Problem.Contest.IsOpen(time.Now())
}

func (s *Submission) Rejudge() error {
	s.resetJudgeSetResults()
	return judge(s.ID)
}

func (s *Submission) SetStatus(status JudgementStatus) error {
	s.Status = status
	err := db.Model(Submission{}).Where("id = ?", s.ID).Update("status", s.Status).Error
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return err
	}

	s.FetchProblem()
	err = onUpdateJudgementStatuses(s.Problem.ContestID, *s)
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
	}
	return err
}

func (s *Submission) Delete() {
	s.FetchJudgeSetResults(false)
	for _, r := range s.JudgeSetResults {
		r.Delete()
	}

	db.Delete(Submission{}, "id = ?", s.ID)
}

func (s *Submission) resetJudgeSetResults() error {
	if err := s.SetStatus(StatusInQueue); err != nil {
		return err
	}

	s.FetchJudgeSetResults(false)
	for i := range s.JudgeSetResults {
		err := s.JudgeSetResults[i].setJudgementStatus(s.Status)
		if err != nil {
			logger.AppLog.Errorf("error: %+v", err)
			return err
		}
	}

	s.FetchProblem()
	onUpdateJudgementStatuses(s.Problem.ContestID, *s)

	return nil
}
