package controllers

import (
	"strconv"

	"github.com/gedorinku/koneko-online-judge/app/models"
	"github.com/revel/revel"
)

type Session struct {
	*revel.Controller
}

func (c Session) Login(email, password string) revel.Result {
	const message = "メールアドレスまたはパスワードが違います。"
	c.Validation.Email(email).Message(message)

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(App.LoginPage)
	}

	session, token, err := models.NewSession(email, password)
	if err != nil {
		c.Validation.Error(message)
	}

	c.Session["userID"] = strconv.Itoa(int(session.User.ID))
	c.Session["sessionToken"] = token

	return c.Redirect(App.Index)
}

func (c Session) Logout() revel.Result {
	userSession := getUserSession(c.Session)
	if userSession != nil {
		userSession.Delete()
	}

	return c.Redirect(App.LoginPage)
}
