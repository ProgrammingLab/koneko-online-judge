package controllers

import (
	"net/http"

	"github.com/ProgrammingLab/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func GetLanguages(c echo.Context) error {
	languages := models.GetAllLanguages()
	return c.JSON(http.StatusOK, languages)
}
