package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Score struct {
	ID           uint          `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Point        int           `json:"point"`
	UserID       uint          `gorm:"not null" json:"userID"`
	ContestID    uint          `gorm:"not null" json:"-"`
	ScoreDetails []ScoreDetail `json:"details"`
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
	found := !db.Where("score_id = ?", s.ID).First(d).RecordNotFound()

	if found && submission.Point <= d.Point && d.Accepted {
		return
	}
	if submission.IsWrong() {
		d.WrongCount += 1
	}
	ac := submission.Status == Accepted
	if found {
		newPoint := MaxInt(d.Point, submission.Point)
		db.Model(d).Updates(map[string]interface{}{"point": newPoint, "wrong_count": d.WrongCount, "accepted": ac})
		db.Model(s).Update("point", s.Point-d.Point+submission.Point)
	} else {
		newScoreDetail(s, submission.ProblemID, submission.Point, d.WrongCount, db)
	}
}

func (s *Score) FetchDetails() {
	db.Model(s).Related(&s.ScoreDetails)
}
