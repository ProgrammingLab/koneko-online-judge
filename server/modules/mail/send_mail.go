package mail

import (
	"github.com/gedorinku/koneko-online-judge/server/conf"
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"gopkg.in/gomail.v2"
)

func SendMail(to, subject, body string) error {
	cfg := conf.GetConfig().SMTP
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	if err := d.DialAndSend(m); err != nil {
		logger.AppLog.Errorf("send mail error: %+v", err)
		return err
	}
	return nil
}
