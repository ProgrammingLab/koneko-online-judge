package models

import (
	"time"
)

type Problem struct {
	ID                uint             `gorm:"primary_key" json:"id"`
	WriterID          uint             `gorm:"not null" json:"writerID"`
	Writer            User             `gorm:"ForeignKey:WriterID" json:"writer;omitempty"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
	Title             string           `gorm:"not null" json:"title"`
	Body              string           `gorm:"type:text; not null" json:"body"`
	InputFormat       string           `gorm:"type:text" json:"inputFormat"`
	OutputFormat      string           `gorm:"type:text" json:"outputFormat"`
	Constraints       string           `gorm:"type:text" json:"constraints"`
	Samples           []Sample         `json:"samples;omitempty"`
	TimeLimit         time.Duration    `gorm:"not null" json:"timeLimit" validate:"required,max=60000000000,min=1000000000"`
	MemoryLimit       int              `gorm:"not null" json:"memoryLimit" validate:"required,max=512,min=128"`
	JudgeType         JudgeType        `gorm:"not null; default:'0'" json:"judgeType" validate:"max=2,min=0"`
	CaseSets          []CaseSet        `json:"caseSets;omitempty"`
	Submissions       []Submission     `json:"-"`
	Contest           *Contest         `json:"contest;omitempty"`
	ContestID         *uint            `json:"contestID"`
	JudgementConfigID *uint            `json:"judgementConfigID;omitempty"`
	JudgementConfig   *JudgementConfig `json:"judgementConfig;omitempty"`
}

type JudgeType int

const (
	// inputとoutputが1対1の普通のジャッジ
	JudgeTypeNormal JudgeType = 0
	// 誤差許容
	JudgeTypePrecision JudgeType = 1
	// 特別なoutputの評価器が必要なジャッジ
	JudgeTypeSpecial JudgeType = 2
)

func NewProblem(problem *Problem) error {
	err := db.Create(problem).Error
	if err != nil {
		return err
	}
	return nil
}

func GetProblem(id uint) *Problem {
	problem := &Problem{}
	notFound := db.Where(id).First(problem).RecordNotFound()
	if notFound {
		return nil
	}
	return problem
}

func GetProblems(contestID *uint, minID, maxID uint, count int) []Problem {
	problems := make([]Problem, 0)
	query := db.Where("contest_id <=> ?", contestID).Where("id >= ?", minID)
	if maxID != 0 {
		query = query.Where("id <= ?", maxID)
	}
	if count != 0 {
		query = query.Limit(count)
	}
	query.Find(&problems)
	return problems
}

func GetNoContestProblems() []Problem {
	problems := make([]Problem, 0)
	db.Order("id ASC").Find(&problems)
	return problems
}

func (p *Problem) Update(request *Problem) {
	defer db.Save(p)
	p.Title = request.Title
	p.TimeLimit = request.TimeLimit
	p.MemoryLimit = request.MemoryLimit
	p.Body = request.Body

	if p.JudgementConfigID == nil && request.JudgementConfig == nil {
		return
	}

	if p.JudgementConfigID != nil {
		db.Delete(&JudgementConfig{}, "id = ?", *p.JudgementConfigID)
	}
	if request.JudgementConfig != nil {
		db.Create(request.JudgementConfig)
		p.JudgementConfigID = &request.JudgementConfig.ID
		p.JudgementConfig = nil
	} else {
		p.JudgementConfigID = nil
		p.JudgementConfig = nil
	}
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
	db.Where("id = ?", p.WriterID).First(&p.Writer)
}

func (p *Problem) FetchCaseSets() {
	db.Model(p).Related(&p.CaseSets)
}

func (p *Problem) FetchSubmissions() {
	db.Model(p).Related(&p.Submissions)
}

func (p *Problem) FetchContest() {
	if p.ContestID == nil {
		return
	}

	p.Contest = &Contest{}
	db.Model(p).Related(p.Contest)
}

func (p *Problem) FetchJudgementConfig() {
	if p.JudgementConfigID == nil {
		return
	}

	p.JudgementConfig = &JudgementConfig{}
	db.Model(p).Related(p.JudgementConfig)
}

func (p *Problem) GetSubmissionsReversed() []Submission {
	submissions := make([]Submission, 0)
	db.Order("id DESC", false).Model(p).Related(&submissions)
	return submissions
}

func (p *Problem) CanView(s *UserSession) bool {
	if p.ContestID == nil {
		return true
	}
	if s == nil {
		return false
	}

	p.FetchContest()
	return p.Contest.CanViewProblems(s)
}

func (p *Problem) CanEdit(s *UserSession) bool {
	if s == nil {
		return false
	}
	if p.ContestID != nil {
		isWriter, _ := IsContestWriter(*p.ContestID, s.UserID)
		return isWriter
	}

	return p.WriterID == s.UserID
}

func (p *Problem) DeleteSamples() {
	p.FetchSamples()
	for _, s := range p.Samples {
		s.Delete()
	}
}

func (p *Problem) Delete() {
	p.DeleteSamples()
	p.FetchSubmissions()
	for _, s := range p.Submissions {
		s.Delete()
	}

	caseSets := make([]CaseSet, 0)
	db.Unscoped().Where("problem_id = ?", p.ID).Find(&caseSets)
	for _, s := range caseSets {
		s.DeletePermanently()
	}

	db.Delete(p)
}
