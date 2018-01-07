package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

var userNotFound = ErrorResponse{"ユーザーが見つかりません"}

func GetMyUser(c echo.Context) error {
	s, _ := c.Get("session").(models.UserSession)
	s.FetchUser()
	return c.JSON(http.StatusOK, s.User)
}

func GetUser(c echo.Context) error {
	user := models.FindUserByName(c.Param("name"))
	if user == nil {
		return c.JSON(http.StatusNotFound, userNotFound)
	}
	user.Email = ""
	return c.JSON(http.StatusOK, user)
}

func GetAllUsers(c echo.Context) error {
	users := models.GetAllUsers()
	for i := range users {
		users[i].Email = ""
	}

	return c.JSON(http.StatusOK, users)
}
