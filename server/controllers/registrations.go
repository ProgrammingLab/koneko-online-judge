package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func StartRegistration(c echo.Context) error {
	req := email{}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	email := models.GetWhiteEmail(req.Email)
	if email == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{"Email Not Found"})
	}

	if err := models.StartEmailConfirmation(email); err != nil {
		return ErrInternalServer
	}

	return c.NoContent(http.StatusNoContent)
}

func VerifyEmailConfirmationToken(c echo.Context) error {
	req := c.Param("token")
	token := models.GetEmailConfirmation(req)
	if token == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{"Invalid Token"})
	}

	return c.NoContent(http.StatusNoContent)
}
