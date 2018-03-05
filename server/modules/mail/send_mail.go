package mail

import (
	"os"
	"strconv"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"gopkg.in/gomail.v2"
)

func SendMail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", getSMTPFrom())
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	p, err := getSMTPPort()
	if err != nil {
		logger.AppLog.Errorf("send mail error: %+v", err)
		return err
	}
	d := gomail.NewDialer(getSMTPHost(), p, getSMTPUser(), getSMTPPass())
	if err := d.DialAndSend(m); err != nil {
		logger.AppLog.Errorf("send mail error: %+v", err)
		return err
	}
	return nil
}

func getSMTPHost() string {
	return os.Getenv("KOJ_SMTP_HOST")
}

func getSMTPPort() (int, error) {
	p, err := strconv.Atoi(os.Getenv("KOJ_SMTP_PORT"))
	return p, err
}

func getSMTPFrom() string {
	return os.Getenv("KOJ_SMTP_FROM")
}

func getSMTPUser() string {
	return os.Getenv("KOJ_SMTP_USER")
}

func getSMTPPass() string {
	return os.Getenv("KOJ_SMTP_PASS")
}
