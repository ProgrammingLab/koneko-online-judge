package models

import (
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type Authority uint

type User struct {
	ID             uint       `gorm:"primary_key" json:"id"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      time.Time  `json:"-"`
	DeletedAt      *time.Time `sql:"index" json:"-"`
	Name           string     `gorm:"not null;unique_index" json:"name"`
	DisplayName    string     `gorm:"not null" json:"displayName"`
	Email          string     `gorm:"not null;unique_index" json:"email,omitempty"`
	Authority      Authority  `gorm:"not null" json:"authority"`
	IsDeleted      bool       `gorm:"not null;default:'0'" json:"-"`
	PasswordDigest string     `gorm:"not null" json:"-"`
}

const (
	Member Authority = 0
	Admin  Authority = 1
)

var ErrUserIDIsZero = errors.New("User IDが0です")

func GetUser(id uint, email bool) *User {
	user := &User{}
	nf := db.Where(id).First(user).RecordNotFound()
	if nf {
		return nil
	}
	if !email {
		user.Email = ""
	}
	return user
}

func GetAllUsers(email bool) []User {
	u := make([]User, 0)
	db.Find(&u)
	if !email {
		for i := range u {
			u[i].Email = ""
		}
	}
	return u
}

func FindUserByName(name string, email bool) *User {
	u := &User{}
	nf := db.Where("name = ?", name).First(u).RecordNotFound()
	if nf {
		return nil
	}
	if !email {
		u.Email = ""
	}
	return u
}

func (u *User) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordDigest), []byte(password))
	return err == nil
}
