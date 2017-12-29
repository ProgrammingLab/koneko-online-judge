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
	initNavigationBar(&c.ViewArgs, c.Session)
	return c.Render()
}

func (c App) LoginPage() revel.Result {
	return c.RenderTemplate("App/Login.html")
}

func getUserSession(session revel.Session) *models.UserSession {
	userID, _ := strconv.Atoi(session[SessionUserIDKey])
	token := session[SessionTokenKey]
	userSession := models.CheckLogin(uint(userID), token)

	return userSession
}

func getUser(session revel.Session) *models.User {
	return &getUserSession(session).User
}

// ViewArgsとSessionを渡すとnavigationに渡す値を入れてくれる
func initNavigationBar(out *map[string]interface{}, session revel.Session) {
	user := getUser(session)
	(*out)["user"] = user
}
