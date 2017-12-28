package controllers

import (
	"github.com/revel/revel"
	"github.com/gedorinku/koneko-online-judge/app/models"
)

type Session struct {
	*revel.Controller
}

func (c Session) Login(email, password string) revel.Result {
	const message = "メールアドレスまたはパスワードが違います。"
	c.Validation.Email(email).Message(message)
	revel.AppLog.Info(password)

	if !c.Validation.HasErrors() {
		_, err := models.NewSession(email, password)
		if err != nil {
			c.Validation.Error(message)
		}
	}

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(App.LoginPage)
	}

	return c.Redirect(App.Index)
}
