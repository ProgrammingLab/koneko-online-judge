package models

import (
	"errors"
	"time"

	"strconv"

	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserSession struct {
	ID          uint `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	User        User
	UserID      uint   `gorm:"not null"`
	TokenDigest string `gorm:"not null"`
}

var (
	LoginError = errors.New("incorrect username or password")

	lifetimeTicks = time.Duration(24 * time.Hour)
)

// emailとpasswordが正しければ新しいUserSessionとTokenを返す
func NewSession(email, password string) (*UserSession, string, error) {
	user := &User{Email: email}
	notFound := db.Where(user).First(user).RecordNotFound()

	if notFound || !user.IsCorrectPassword(password) {
		return nil, "", LoginError
	}

	secret := []byte(GenerateRandomBase64String(24))
	digest, err := bcrypt.GenerateFromPassword(secret, bcrypt.MinCost)
	if err != nil {
		return nil, "", err
	}

	oldSession := getSessionFromUser(user.ID)
	if oldSession != nil {
		db.Delete(oldSession)
	}
	session := &UserSession{
		User:        *user,
		TokenDigest: string(digest),
	}
	err = db.Create(session).Error
	if err != nil {
		return nil, "", err
	}

	token := strconv.Itoa(int(session.ID)) + "_" + string(secret)

	return session, token, nil
}

func CheckLogin(token string) *UserSession {
	tokens := strings.Split(token, "_")
	if len(tokens) != 2 {
		return nil
	}
	id, err := strconv.Atoi(tokens[0])
	if err != nil {
		return nil
	}
	session := GetSession(uint(id))
	if session == nil {
		return nil
	}
	duration := time.Now().Sub(session.CreatedAt)
	if lifetimeTicks < duration {
		return nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(session.TokenDigest), []byte(tokens[1]))
	if err != nil {
		return nil
	}

	session.FetchUser()
	return session
}

func GetSession(id uint) *UserSession {
	s := &UserSession{}
	nf := db.Where(id).First(s).RecordNotFound()
	if nf {
		return nil
	}
	return s
}

func getSessionFromUser(userID uint) *UserSession {
	session := &UserSession{UserID: userID}
	notFound := db.Where(session).First(session).RecordNotFound()
	if notFound {
		return nil
	}
	return session
}

func (s *UserSession) Delete() {
	db.Delete(s)
	s.TokenDigest = GenerateRandomBase64String(16)
}

func (s *UserSession) FetchUser() {
	db.Model(s).Related(&s.User)
}
