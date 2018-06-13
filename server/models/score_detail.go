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

func newScoreDetail(score *Score, submission *Submission, wrongCount int, tx *gorm.DB) *ScoreDetail {
	d := &ScoreDetail{
		Point:      submission.Point,
		WrongCount: wrongCount,
		Accepted:   submission.Status == StatusAccepted,
		ScoreID:    score.ID,
		ProblemID:  submission.ProblemID,
	}
	tx.Create(d)
	tx.Model(d).UpdateColumns(map[string]interface{}{
		"created_at": submission.CreatedAt,
		"updated_at": submission.CreatedAt,
	})

	return d
}

func deleteScoreDetails(problemID uint) {
	db.Delete(ScoreDetail{}, "problem_id = ?", problemID)
}
