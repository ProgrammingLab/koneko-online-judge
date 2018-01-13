package controllers

import (
	"strconv"

	"net/http"

	"io/ioutil"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func NewProblem(c echo.Context) error {
	problem := &models.Problem{}
	if err := c.Bind(problem); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(problem); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	s := c.Get("session").(models.UserSession)
	s.FetchUser()
	problem.ID = 0
	problem.Contest = nil
	problem.WriterID = s.ID
	problem.Writer = s.User
	if err := models.NewProblem(problem); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
	}
	if problem.ContestID == nil {
		problem.ContestID = new(uint)
	}
	return c.JSON(http.StatusCreated, problem)
}

func UpdateProblem(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.ErrNotFound
	}

	request := &models.Problem{}
	if err := c.Bind(request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	request.WriterID = 0
	request.Writer = models.User{}

	problem := models.GetProblem(uint(id))
	if problem == nil {
		return echo.ErrNotFound
	}

	problem.ContestID = nil
	problem.Contest = nil
	problem.DeleteSamples()
	problem.Update(request)
	return c.NoContent(http.StatusNoContent)
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
	problem.FetchCaseSets()
	problem.FetchContest()
	if problem.ContestID == nil {
		problem.ContestID = new(uint)
	}

	s := c.Get("session").(models.UserSession)
	if problem.WriterID != s.UserID {
		problem.JudgeSourceCode = ""
	}

	return c.JSON(http.StatusOK, problem)
}

func UpdateCases(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.ErrNotFound
	}

	problem := models.GetProblem(uint(id))
	if problem == nil {
		return echo.ErrNotFound
	}

	buf, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	err = problem.ReplaceTestCases(buf)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	problem.FetchCaseSets()

	return c.JSON(http.StatusOK, problem.CaseSets)
}
