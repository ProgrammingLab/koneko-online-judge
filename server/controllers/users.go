package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

var userNotFound = ErrorResponse{"ユーザーが見つかりません"}

func GetMyUser(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return echo.ErrNotFound
	}
	s.FetchUser()
	return c.JSON(http.StatusOK, s.User)
}

func GetUser(c echo.Context) error {
	user := models.FindUserByName(c.Param("name"), false)
	if user == nil {
		return c.JSON(http.StatusNotFound, userNotFound)
	}
	return c.JSON(http.StatusOK, user)
}

func GetAllUsers(c echo.Context) error {
	users := models.GetAllUsers(false)

	return c.JSON(http.StatusOK, users)
}
