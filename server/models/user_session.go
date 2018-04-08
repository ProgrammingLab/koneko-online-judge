package models

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserSession struct {
	ID          uint   `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	User        User
	UserID      uint   `gorm:"not null"`
	TokenDigest string `gorm:"not null"`
}

const retryLimit = 100

var (
	ErrLogin               = errors.New("incorrect username or password")
	ErrSessionCreateFailed = errors.New("retry limit exceeded")
	errSessionIDDuplicated = errors.New("session id is duplicated")

	lifetimeTicks = time.Duration(24 * time.Hour)
	maxID         = big.NewInt(math.MaxUint32 - 1)
)

// emailとpasswordが正しければ新しいUserSessionとTokenを返す
func NewSession(email, password string) (*UserSession, string, error) {
	user := &User{Email: email}
	notFound := db.Where(user).First(user).RecordNotFound()

	if notFound || !user.IsCorrectPassword(password) {
		return nil, "", ErrLogin
	}

	secret := []byte(GenerateRandomBase64String(24))
	digest, err := bcrypt.GenerateFromPassword(secret, bcrypt.MinCost)
	if err != nil {
		return nil, "", err
	}

	session := &UserSession{
		UserID:      user.ID,
		TokenDigest: string(digest),
	}
	for i := 0; i < retryLimit; i++ {
		err := tryCreateSession(session)
		if err == nil {
			break
		}
	}

	session.User = *user
	token := strconv.Itoa(int(session.ID)) + "_" + string(secret)

	return session, token, nil
}

func tryCreateSession(session *UserSession) error {
	bn, err := rand.Int(rand.Reader, maxID)
	if err != nil {
		return err
	}
	n := uint(bn.Int64()) + 1
	nf := db.Table("user_sessions").Where("id = ?", n).Scan(&UserSession{}).RecordNotFound()
	if !nf {
		return errSessionIDDuplicated
	}

	session.ID = n
	return db.Create(session).Error
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
		session.Delete()
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

func (s *UserSession) Delete() {
	db.Delete(UserSession{}, "id = ?", s.ID)
	s.TokenDigest = GenerateRandomBase64String(16)
}

func (s *UserSession) FetchUser() {
	db.Model(s).Related(&s.User)
}
