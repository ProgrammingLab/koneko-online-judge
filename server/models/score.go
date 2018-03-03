package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Score struct {
	ID           uint      `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Point        int       `json:"point"`
	UserID       uint      `gorm:"not null"`
	ContestID    uint      `gorm:"not null"`
	ScoreDetails []ScoreDetail
}

func newScore(userID, contestID uint, tx *gorm.DB) *Score {
	s := &Score{
		Point:     0,
		UserID:    userID,
		ContestID: contestID,
	}
	tx.Create(s)

	return s
}

func updateScore(submission *Submission, contestID uint) {
	s := &Score{}
	db.Where("user_id = ? AND contest_id = ?", submission.UserID, contestID).First(s)

	d := &ScoreDetail{}
	found := !db.Where("score_id = ?").First(d).RecordNotFound()

	d.Point = submission.Point
	if submission.IsWrong() {
		d.WrongCount += 1
	}
	if found {
		db.Model(d).Updates(map[string]interface{}{"point": d.Point, "wrong_count": d.WrongCount})
	} else {
		newScoreDetail(s, submission.ProblemID, d.Point, d.WrongCount, db)
	}
}

func (s *Score) FetchDetails() {
	db.Model(s).Related(&s.ScoreDetails)
}
