package controllers

import (
	"strings"

	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func CheckLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Path() == "/sessions/login" {
			return next(c)
		}

		const bearer = "Bearer"
		auth := c.Request().Header.Get("Authorization")
		values := strings.Split(auth, " ")
		if len(values) < 2 || values[0] != bearer {
			return c.JSON(http.StatusBadRequest, ErrorResponse{"Authorizationヘッダーが不正です"})
		}

		s := models.CheckLogin(values[1])
		if s == nil {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{"認証に失敗しました"})
		}

		c.Set("session", *s)

		return next(c)
	}
}
