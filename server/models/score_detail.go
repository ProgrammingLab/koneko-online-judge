package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type ScoreDetail struct {
	ID         uint      `gorm:"primary_key" json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Point      int       `json:"point"`
	WrongCount int       `gorm:"not null" json:"wrongCount"`
	Accepted   bool      `gorm:"not null" json:"accepted"`
	ScoreID    uint      `gorm:"not null" json:"-"`
	ProblemID  uint      `gorm:"not null" json:"problemID"`
}

func newScoreDetail(score *Score, problemID uint, point, wrongCount int, tx *gorm.DB) *ScoreDetail {
	d := &ScoreDetail{
		Point:      point,
		WrongCount: wrongCount,
		ScoreID:    score.ID,
		ProblemID:  problemID,
	}
	tx.Create(d)

	tx.Model(score).Update("point", score.Point+point)

	return d
}
