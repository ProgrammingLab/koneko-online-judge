package controllers

import (
	"strconv"

	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func NewProblem(c echo.Context) error {
	return nil
}

func GetProblem(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.ErrNotFound
	}

	problem := models.GetProblem(uint(id))
	if problem == nil {
		return echo.ErrNotFound
	}

	problem.FetchWriter()
	problem.Writer.Email = ""
	problem.FetchSamples()

	return c.JSON(http.StatusOK, problem)
}
