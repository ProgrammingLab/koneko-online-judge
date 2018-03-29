package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

type password struct {
	Password string `json:"password" validate:"required"`
}

func SendPasswordResetMail(c echo.Context) error {
	email := &email{}
	if err := c.Bind(email); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(email); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	u := models.FindUserByEmail(email.Email)
	if u == nil {
		return echo.ErrNotFound
	}

	if err := models.StartPasswordReset(u); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{"Internal Server Error"})
	}

	return c.NoContent(http.StatusNoContent)
}

func VerifyPasswordResetToken(c echo.Context) error {
	token := c.Param("token")
	if t := models.GetPasswordResetToken(token); t == nil {
		return echo.ErrNotFound
	}
	return c.NoContent(http.StatusNoContent)
}

func ResetPassword(c echo.Context) error {
	token := c.Param("token")
	pass := &password{}

	if err := c.Bind(pass); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(pass); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	switch models.ResetPassword(token, pass.Password) {
	case nil:
		return c.NoContent(http.StatusNoContent)
	case models.ErrInvalidToken:
		return echo.ErrNotFound
	default:
		return ErrInternalServer
	}
}
