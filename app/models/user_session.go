package models

import (
	"errors"
	"time"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

type UserSession struct {
	gorm.Model
	TokenDigest   string `gorm:"not null"`
	LifetimeTicks int64  `gorm:"not null"`
}

var (
	errorLogin = errors.New("incorrect username or password")

	lifetimeTicks = time.Duration(24 * time.Hour)
)

func NewSession(email, password string) (*UserSession, error) {
	user := &User{Email: email}
	db.Where(user).First(user)

	if user.ID == 0 {
		return nil, errorLogin
	}
	if !user.IsCorrectPassword(password) {
		return nil, errorLogin
	}

	token := []byte(GenerateSecretToken(32))
	digest, _ := bcrypt.GenerateFromPassword(token, GetBcryptCost())

	session := &UserSession{
		TokenDigest:   string(digest),
		LifetimeTicks: int64(lifetimeTicks),
	}
	db.Create(session)

	return session, nil
}
