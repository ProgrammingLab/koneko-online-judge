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

var responseUnauthorized = ErrorResponse{"Unauthorized"}

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
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
	}
	s.Delete()
	return c.NoContent(http.StatusNoContent)
}

func getSession(c echo.Context) *models.UserSession {
	s, ok := c.Get("session").(models.UserSession)
	if ok {
		return &s
	}
	return nil
}
