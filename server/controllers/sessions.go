package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

type sessionRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	Token string `json:"token"`
}

func Login(c echo.Context) error {
	r := &sessionRequest{}
	err := c.Bind(r)
	if err != nil {
		return err
	}

	_, token, err := models.NewSession(r.Email, r.Password)
	if err != nil {
		if err == models.ErrLogin {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{"メールアドレスかパスワードが間違っています"})
		}

		return c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
	}

	return c.JSON(http.StatusCreated, sessionResponse{token})
}

func Logout(c echo.Context) error {
	s, _ := c.Get("session").(models.UserSession)
	s.Delete()
	return c.NoContent(http.StatusNoContent)
}

func getSession(c echo.Context) models.UserSession {
	return c.Get("session").(models.UserSession)
}
