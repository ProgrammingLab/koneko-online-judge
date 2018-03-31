package models

import (
	"fmt"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/modules/mail"
	"github.com/jinzhu/gorm"
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

	subjectPasswordChanged = "[Koneko Online Judge]パスワードが変更されました"
	bodyPasswordChanged    = `<p>%v さん</p>
<p>お使いのKoneko Online Judgeアカウントのパスワードが変更されましたので、お知らせいたします。</p>`
)

var (
	ErrUserIDIsZero          = errors.New("User IDが0です")
	ErrUserNameAlreadyExists = errors.New("user nameはすでに使用されています")
	ErrEmailAlreadyExists    = errors.New("emailはすでに使用されています")
)

func NewUser(name, displayName, email, password string, token *EmailConfirmation) (*User, error) {
	u := &User{
		Name:        name,
		DisplayName: displayName,
		Email:       email,
	}

	tx := db.Begin()
	nf := tx.Model(User{}).Where("name = ?", name).Limit(1).Scan(&User{}).RecordNotFound()
	if !nf {
		tx.Rollback()
		return nil, ErrUserNameAlreadyExists
	}
	nf = tx.Model(User{}).Where("email = ?", email).Limit(1).Scan(&User{}).RecordNotFound()
	if !nf {
		tx.Rollback()
		return nil, ErrEmailAlreadyExists
	}

	if err := tx.Create(u).Error; err != nil {
		tx.Rollback()
		logger.AppLog.Errorf("error: %+v", err)
		return nil, err
	}

	if err := u.setPasswordWithinTransaction(tx, password, false); err != nil {
		tx.Rollback()
		logger.AppLog.Errorf("error: %+v", err)
		return nil, err
	}

	err := tx.Where("id = ?", token.ID).Delete(EmailConfirmation{}).Error
	if err != nil {
		tx.Rollback()
		logger.AppLog.Errorf("error: %+v", err)
		return nil, err
	}

	err = tx.Where("email = ?", email).Delete(WhiteEmail{}).Error
	if err != nil {
		tx.Rollback()
		logger.AppLog.Errorf("error: %+v", err)
		return nil, err
	}

	tx.Commit()
	return u, nil
}

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

func FindUserByEmail(email string) *User {
	u := &User{}
	nf := db.Where("email = ?", email).First(u).RecordNotFound()
	if nf {
		return nil
	}
	return u
}

func (u *User) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordDigest), []byte(password))
	return err == nil
}

func (u *User) SetPassword(password string, notification bool) error {
	return u.setPasswordWithinTransaction(db, password, notification)
}

func (u *User) setPasswordWithinTransaction(tx *gorm.DB, password string, notification bool) error {
	d, err := bcrypt.GenerateFromPassword([]byte(password), GetBcryptCost())
	if err != nil {
		return err
	}

	u.PasswordDigest = string(d)
	err = tx.Model(u).Update("password_digest", string(d)).Error
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return err
	}

	if !notification {
		return nil
	}

	body := fmt.Sprintf(bodyPasswordChanged, u.Name)
	err = mail.SendMail(u.Email, subjectPasswordChanged, body)
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
	}
	return err
}

func (u *User) IsAdmin() bool {
	return u.Authority == Admin
}
