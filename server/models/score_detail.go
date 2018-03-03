package models

import "time"

type ScoreDetail struct {
	ID         uint      `gorm:"primary_key" json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Point      int       `json:"point"`
	WrongCount int       `gorm:"not null" json:"wrongCount"`
	ScoreID    uint      `gorm:"not null" json:"-"`
	ProblemID  uint      `gorm:"not null" json:"problemID"`
}

func NewScoreDetail(score *Score, problemID uint, point int) *ScoreDetail {
	d := &ScoreDetail{
		Point:      point,
		WrongCount: 0,
		ScoreID:    score.ID,
		ProblemID:  problemID,
	}
	db.Create(d)

	db.Model(score).Update("point", score.Point+point)

	return d
}
