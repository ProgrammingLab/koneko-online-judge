package models

import (
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	gorm.Model
	Name           string `gorm:"not null;unique_index"`
	DisplayName    string `gorm:"not null"`
	Email          string `gorm:"not null;unique_index"`
	Authority      uint   `gorm:"not null"`
	IsDeleted      bool   `gorm:"not null;default:'0'"`
	PasswordDigest string `gorm:"not null"`
}

const authorityMember = 0
const authorityAdmin = 1

func (u *User) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordDigest), []byte(password))
	return err == nil
}
