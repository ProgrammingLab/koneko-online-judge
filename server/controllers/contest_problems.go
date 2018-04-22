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

func GetContestProblems(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return echo.ErrUnauthorized
	}

	contest := getContestFromContext(c)
	if contest == nil || !contest.CanViewProblems(s) {
		return echo.ErrNotFound
	}

	contest.FetchProblems()
	for i := range contest.Problems {
		fetchProblem(&contest.Problems[i], s)
	}

	return c.JSON(http.StatusOK, contest.Problems)
}
