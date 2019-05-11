package nekomail

import (
	"time"

	"github.com/ProgrammingLab/koneko-online-judge/server/conf"
	"github.com/ProgrammingLab/koneko-online-judge/server/logger"
	"github.com/go-mail/mail"
)

func SendMail(to, subject, body string) error {
	cfg := conf.GetConfig().SMTP
	m := mail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	var tls mail.StartTLSPolicy
	if cfg.NoStartTLS {
		tls = mail.NoStartTLS
	} else {
		tls = mail.OpportunisticStartTLS
	}

	d := &mail.Dialer{
		Host:           cfg.Host,
		Port:           cfg.Port,
		Username:       cfg.User,
		Password:       cfg.Password,
		SSL:            cfg.Port == 465,
		Timeout:        10 * time.Second,
		RetryFailure:   true,
		StartTLSPolicy: tls,
	}

	if err := d.DialAndSend(m); err != nil {
		logger.AppLog.Errorf("send mail error: %+v", err)
		return err
	}
	return nil
}
