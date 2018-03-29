package models

import (
	"fmt"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/modules/mail"
)

type EmailConfirmation struct {
	ID           uint          `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	LifeTime     time.Duration `json:"lifeTime"`
	Token        string        `gorm:"not null" json:"-"`
	WhiteEmailID uint          `gorm:"not null" json:"whiteEmailID"`
	WhiteEmail   WhiteEmail    `json:"whiteEmail"`
}

const (
	EmailConfirmationTokenLifetime = 30 * time.Minute

	subjectEmailConfirmation = "[Koneko Online Judge]ユーザー登録用リンク"
	bodyEmailConfirmation    = `<p>Koneko Online Judgeのユーザー登録がリクエストされました。
ユーザー登録を行うには、下記のリンクをクリックしてください。</p>
<p><a href="https://judge.kurume-nct.com/#/registration/%v">https://judge.kurume-nct.com/#/registration/%v</a></p>`
)

func StartEmailConfirmation(email *WhiteEmail) error {
	if email == nil {
		return ErrNilArgument
	}

	confirm, err := newEmailConfirmation(email)
	if err != nil {
		return err
	}

	body := fmt.Sprintf(bodyEmailConfirmation, confirm.Token, confirm.Token)
	return mail.SendMail(email.Email, subjectEmailConfirmation, body)
}

func newEmailConfirmation(email *WhiteEmail) (*EmailConfirmation, error) {
	db.Delete(EmailConfirmation{}, "white_email_id = ?", email.ID)

	token, err := GenerateRandomBase62String(48)
	if err != nil {
		logger.AppLog.Errorf("token error: %+v", err)
		return nil, err
	}

	c := &EmailConfirmation{
		LifeTime:     EmailConfirmationTokenLifetime,
		Token:        token,
		WhiteEmailID: email.ID,
	}
	if err := db.Create(c).Error; err != nil {
		logger.AppLog.Errorf("insert error: %+v", err)
		return nil, err
	}

	return c, nil
}
