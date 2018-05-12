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
	Token        string        `gorm:"not null; index" json:"-"`
	WhiteEmailID uint          `gorm:"not null" json:"whiteEmailID"`
	WhiteEmail   WhiteEmail    `json:"whiteEmail"`
}

const (
	EmailConfirmationTokenLifetime = 7 * 24 * time.Hour

	subjectEmailConfirmation = "[Koneko Online Judge]招待のご案内"
	bodyEmailConfirmation    = `<p>Koneko Online Judgeへ招待されました。
ユーザー登録を行うには、下記のリンクをクリックしてください。</p>
<p><a href="https://judge.kurume-nct.com/#/registration/%v">https://judge.kurume-nct.com/#/registration/%v</a></p>
<p>このリンクの有効期間は1週間です。</p>`
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

func GetEmailConfirmation(token string) *EmailConfirmation {
	res := &EmailConfirmation{}
	nf := db.Model(EmailConfirmation{}).Where("token = ?", token).Scan(res).RecordNotFound()
	if nf {
		return nil
	}
	if time.Now().After(res.CreatedAt.Add(res.LifeTime)) {
		db.Where("id = ?", res.ID).Delete(EmailConfirmation{})
		return nil
	}

	return res
}

func (c *EmailConfirmation) FetchWhiteEmail() {
	res := WhiteEmail{}
	db.Model(res).Where("id = ?", c.WhiteEmailID).Scan(&res)
	c.WhiteEmail = res
}
