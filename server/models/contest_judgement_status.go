package models

import "time"

type ContestJudgementStatus struct {
	ContestID uint            `gorm:"primary_key" sql:"type:int unsigned" json:"contestID"`
	UserID    uint            `gorm:"primary_key" sql:"type:int unsigned" json:"userID"`
	ProblemID uint            `gorm:"not null" json:"problemID"`
	UpdatedAt time.Time       `json:"updatedAt"`
	Status    JudgementStatus `gorm:"not null; default:'0'" json:"status"`
	Point     int             `gorm:"not null; default:'0'" json:"point"`
}
