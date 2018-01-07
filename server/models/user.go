package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             uint       `gorm:"primary_key" json:"-"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      time.Time  `json:"-"`
	DeletedAt      *time.Time `sql:"index" json:"-"`
	Name           string     `gorm:"not null;unique_index" json:"name"`
	DisplayName    string     `gorm:"not null" json:"displayName"`
	Email          string     `gorm:"not null;unique_index" json:"email,omitempty"`
	Authority      uint       `gorm:"not null" json:"authority"`
	IsDeleted      bool       `gorm:"not null;default:'0'" json:"-"`
	PasswordDigest string     `gorm:"not null" json:"-"`
}

const authorityMember = 0
const authorityAdmin = 1

func GetAllUsers() []User {
	u := make([]User, 0)
	db.Find(&u)
	return u
}

func FindUserByName(name string) *User {
	u := &User{}
	nf := db.Where("name = ?", name).First(u).RecordNotFound()
	if nf {
		return nil
	}
	return u
}

func (u *User) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordDigest), []byte(password))
	return err == nil
}
