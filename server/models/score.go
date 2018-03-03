package models

import "time"

type Score struct {
	ID           uint      `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Point        int       `json:"point"`
	UserID       uint      `gorm:"not null"`
	ContestID    uint      `gorm:"not null"`
	ScoreDetails []ScoreDetail
}

func NewScore(userID, contestID uint) *Score {
	s := &Score{
		Point:     0,
		UserID:    userID,
		ContestID: contestID,
	}
	db.Create(s)

	return s
}

func (s *Score) FetchDetails() {
	db.Model(s).Related(&s.ScoreDetails)
}
