package models

import (
	"time"
)

type Problem struct {
	ID              uint          `gorm:"primary_key"`
	WriterID        uint          `gorm:"not null"`
	Writer          User
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Title           string        `gorm:"not null"`
	Body            string        `gorm:"type:text; not null"`
	Samples         []Sample
	TimeLimit       time.Duration `gorm:"not null"`
	MemoryLimit     int           `gorm:"not null"`
	JudgeType       int           `gorm:"not null; default:'0'"`
	JudgeSourceCode string        `gorm:"type:text"`
	CaseSets        []CaseSet
}

const (
	// inputとoutputが1対1の普通のジャッジ
	JudgeTypeNormal = 0
	// 誤差許容
	JudgeTypePrecision = 1
	// 特別なoutputの評価器が必要なジャッジ
	JudgeTypeSpecial = 2
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
