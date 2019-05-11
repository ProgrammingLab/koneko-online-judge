package models

import (
	"fmt"
	"time"

	"github.com/ProgrammingLab/koneko-online-judge/server/logger"
	"github.com/ProgrammingLab/koneko-online-judge/server/modules/nekomail"
	"github.com/ProgrammingLab/koneko-online-judge/server/modules/unique"
	"github.com/pkg/errors"
)

const (
	subjectPasswordReset = "[Koneko Online Judge]パスワード再設定用リンク"
	bodyPasswordReset    = `<p>%v さん</p>
<p>Koneko Online Judgeのパスワード再設定がリクエストされました。
パスワードを再設定するには、下記のリンクをクリックしてください。</p>
<p><a href="https://judge.kurume-nct.com/password_reset/%v">https://judge.kurume-nct.com/password_reset/%v</a></p>`
)

type PasswordResetToken struct {
	ID        uint      `gorm:"primary_key" json:"-"`
	CreatedAt time.Time `json:"-"`
	Token     string    `gorm:"index; not null"`
	UserID    uint      `gorm:"not null"`
	User      User
}

const PasswordResetTokenLifeTime = 30 * time.Minute

var ErrInvalidToken = errors.New("invalid password reset token")

func StartPasswordReset(user *User) error {
	t, err := newPasswordResetToken(user)
	if err != nil {
		return nil
	}

	body := fmt.Sprintf(bodyPasswordReset, user.Name, t.Token, t.Token)
	return nekomail.SendMail(user.Email, subjectPasswordReset, body)
}

func newPasswordResetToken(user *User) (*PasswordResetToken, error) {
	db.Delete(PasswordResetToken{}, "user_id = ?", user.ID)
	token, err := unique.GenerateRandomBase62String(48)
	if err != nil {
		return nil, err
	}
	t := &PasswordResetToken{
		Token:  token,
		UserID: user.ID,
	}
	if err := db.Create(t).Error; err != nil {
		return nil, err
	}

	return t, nil
}

func GetPasswordResetToken(token string) *PasswordResetToken {
	t := &PasswordResetToken{}
	if db.Where("token = ?", token).First(t).RecordNotFound() {
		return nil
	}
	if PasswordResetTokenLifeTime < time.Now().Sub(t.CreatedAt) {
		db.Delete(PasswordResetToken{}, "id = ?", t.ID)
		return nil
	}
	return t
}

func ResetPassword(token, password string) error {
	t := GetPasswordResetToken(token)
	if t == nil {
		return ErrInvalidToken
	}

	t.FetchUser()
	defer db.Delete(PasswordResetToken{}, "id = ?", t.ID)
	if err := t.User.SetPassword(password, true); err != nil {
		logger.AppLog.Errorf("err %+v", err)
		return err
	}
	return nil
}

func (t *PasswordResetToken) FetchUser() {
	db.Model(t).Related(&t.User)
}
