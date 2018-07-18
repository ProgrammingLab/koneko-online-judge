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
	// TODO コンテスト開始時間を変更したあとにリジャッジすると、合計点がバグる
	s := &Score{}
	db.Where("user_id = ? AND contest_id = ?", submission.UserID, contestID).First(s)

	d := &ScoreDetail{}
	found := !db.Where("score_id = ? AND problem_id = ?", s.ID, submission.ProblemID).First(d).RecordNotFound()

	if found && submission.Point <= d.Point && d.Accepted {
		return
	}
	if submission.IsWrong() {
		d.WrongCount += 1
	}
	ac := submission.Status == StatusAccepted
	if found {
		newPoint := MaxInt(d.Point, submission.Point)
		db.Model(d).UpdateColumns(map[string]interface{}{
			"updated_at":  submission.CreatedAt,
			"point":       newPoint,
			"wrong_count": d.WrongCount,
			"accepted":    ac,
		})
	} else {
		newScoreDetail(s, submission, d.WrongCount, db)
	}

	s.FetchDetails()
	s.Point = 0
	for i := range s.ScoreDetails {
		s.Point += s.ScoreDetails[i].Point
	}
	db.Model(s).UpdateColumns(map[string]interface{}{"point": s.Point, "updated_at": submission.CreatedAt})
}

func (s *Score) FetchDetails() {
	db.Model(s).Related(&s.ScoreDetails)
}
