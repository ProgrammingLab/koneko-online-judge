package models

import "time"

type Problem struct {
	ID              uint          `gorm:"primary_key"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Title           string        `gorm:"not null"`
	Body            string        `gorm:"type:text; not null"`
	Samples         []Sample
	TimeLimit       time.Duration `gorm:"not null"`
	MemoryLimit     int           `gorm:"not null"`
	JudgeType       int           `gorm:"not null; default:'0'"`
	JudgeSourceCode string        `gorm:"type:text"`
}

const (
	// inputとoutputが1対1の普通のジャッジ
	JudgeTypeNormal = 0
	// 誤差許容
	JudgeTypePrecision = 1
	// 特別なoutputの評価器が必要なジャッジ
	JudgeTypeSpecial = 2
)

func (p *Problem) FetchSamples() {
	db.Model(p).Related(&p.Samples)
}
