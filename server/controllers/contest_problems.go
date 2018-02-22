package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func NewContestProblem(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
	}

	contestID, err := getContestIDFromContext(c)
	if err != nil {
		return echo.ErrNotFound
	}
	if !models.CanEditContest(contestID, s.UserID) {
		return echo.ErrNotFound
	}

	return NewProblem(c)
}
