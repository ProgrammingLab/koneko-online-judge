package controllers

import (
	"github.com/revel/revel"
	"github.com/gedorinku/koneko-online-judge/app/models"
	"strconv"
)

const (
	SessionUserIDKey = "userID"
	SessionTokenKey  = "sessionToken"
)

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	user := getUser(c.Session)
	revel.AppLog.Info(user.Name)
	return c.Render(user)
}

func (c App) LoginPage() revel.Result {
	return c.RenderTemplate("App/Login.html")
}

func getUser(session revel.Session) *models.User {
	userID, _ := strconv.Atoi(session[SessionUserIDKey])
	token := session[SessionTokenKey]
	user := models.CheckLogin(uint(userID), token)
	if user == nil {
		// この時点でログインできてないのは多分バグ
		revel.AppLog.Fatal("多分バグ: getUser == nil")
	}

	return user
}
