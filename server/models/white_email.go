package models

import "time"

type WhiteEmail struct {
	ID          uint          `gorm:"primary_key" json:"id"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	LifeTime    time.Duration `json:"lifeTime"`
	Email       string        `gorm:"not null" json:"email"`
	CreatedByID uint          `gorm:"not null" json:"createdByID"`
	CreatedBy   User          `json:"createdBy"`
}
