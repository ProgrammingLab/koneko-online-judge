package models

import "time"

type EmailConfirmation struct {
	ID           uint          `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	LifeTime     time.Duration `json:"lifeTime"`
	Token        string        `gorm:"not null" json:"-"`
	WhiteEmailID uint          `gorm:"not null" json:"whiteEmailID"`
	WhiteEmail   WhiteEmail    `json:"whiteEmail"`
}
