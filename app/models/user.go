package models

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model
	Name        string `gorm:"unique_index"`
	DisplayName string `gorm:"not null"`
	Email       string `gorm:"not null"`
	Authority   uint   `gorm:"not null"`
	IsDeleted   bool   `gorm:"not null;default:'0'"`
	PasswordDigest string `gorm:"not null"`
	RememberTokenDigest string
}

const AUTHORITY_MEMBER = 0
const AUTHORITY_ADMIN = 1
