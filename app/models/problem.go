package models

import (
	"time"
)

type Problem struct {
	ID              uint `gorm:"primary_key"`
	WriterID        uint `gorm:"not null"`
	Writer          User
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Title           string `gorm:"not null"`
	Body            string `gorm:"type:text; not null"`
	InputFormat     string `gorm:"type:text"`
	OutputFormat    string `gorm:"type:text"`
	Constraints     string `gorm:"type:text"`
	Samples         []Sample
	TimeLimit       time.Duration `gorm:"not null"`
	MemoryLimit     int           `gorm:"not null"`
	JudgeType       int           `gorm:"not null; default:'0'"`
	JudgeSourceCode string        `gorm:"type:text"`
	CaseSets        []CaseSet
	Submissions     []Submission
}

type JudgeType int

const (
	// inputとoutputが1対1の普通のジャッジ
	Normal JudgeType = 0
	// 誤差許容
	Precision JudgeType = 1
	// 特別なoutputの評価器が必要なジャッジ
	Special JudgeType = 2
)

func NewProblem(user *User) *Problem {
	problem := &Problem{
		WriterID:    user.ID,
		TimeLimit:   time.Second,
		MemoryLimit: 128,
	}
	db.Create(problem)
	return problem
}

func GetProblem(id uint) *Problem {
	problem := &Problem{}
	db.Where(id).First(problem)
	if problem.ID == 0 {
		return nil
	}
	return problem
}

func GetNoContestProblems() []Problem {
	problems := make([]Problem, 0)
	db.Find(&problems)
	return problems
}

func (p *Problem) Update(request *Problem) {
	p.Title = request.Title
	p.TimeLimit = request.TimeLimit
	p.MemoryLimit = request.MemoryLimit
	p.Body = request.Body
	db.Save(p)
}

func (p *Problem) ReplaceTestCases(archive []byte) error {
	p.FetchCaseSets()
	for _, c := range p.CaseSets {
		c.Delete()
	}

	_, err := newCaseSets(p, archive)
	return err
}

func (p *Problem) FetchSamples() {
	db.Model(p).Related(&p.Samples)
}

func (p *Problem) FetchWriter() {
	db.Model(p).Related(&p.Writer)
}

func (p *Problem) FetchCaseSets() {
	db.Model(p).Related(&p.CaseSets)
}

func (p *Problem) FetchSubmissions() {
	db.Model(p).Related(&p.Submissions)
}

func (p *Problem) CanView(user *User) bool {
	if user == nil {
		return false
	}
	// TODO コンテストの問題だったらその辺を考える
	return true
}
