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

	s := c.Get("session").(models.UserSession)
	problem := models.GetProblem(uint(id))
	if problem == nil || problem.CanView(s.UserID) {
		return echo.ErrNotFound
	}

	problem.ContestID = nil
	problem.Contest = nil
	problem.DeleteSamples()
	problem.Update(request)
	return c.NoContent(http.StatusNoContent)
}

func GetProblems(c echo.Context) error {
	minID, err := strconv.Atoi(models.DefaultString(c.QueryParam("minID"), "0"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	maxID, err := strconv.Atoi(models.DefaultString(c.QueryParam("maxID"), "0"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	count, err := strconv.Atoi(models.DefaultString(c.QueryParam("count"), "0"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if minID < 0 || maxID < 0 || count < 0 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"0以上の値を指定してください"})
	}

	s := c.Get("session").(models.UserSession)

	problems := models.GetProblems(nil, uint(minID), uint(maxID), count)
	for i := range problems {
		if !problems[i].CanView(s.UserID) {
			return echo.ErrNotFound
		}
		fetchProblem(&problems[i], s.UserID)
	}

	return c.JSON(http.StatusOK, problems)
}

func GetProblem(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.ErrNotFound
	}

	s := c.Get("session").(models.UserSession)
	problem := models.GetProblem(uint(id))
	if problem == nil || !problem.CanView(s.UserID) {
		return echo.ErrNotFound
	}

	fetchProblem(problem, s.UserID)

	return c.JSON(http.StatusOK, problem)
}

func UpdateCases(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.ErrNotFound
	}

	s := c.Get("session").(models.UserSession)
	problem := models.GetProblem(uint(id))
	if problem == nil || !problem.CanEdit(s.UserID) {
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

func fetchProblem(out *models.Problem, userID uint) {
	out.FetchWriter()
	out.FetchSamples()
	out.FetchCaseSets()
	out.FetchContest()

	out.Writer.Email = ""
	if out.ContestID == nil {
		out.ContestID = new(uint)
	}

	if out.WriterID != userID {
		out.JudgeSourceCode = ""
	}
}
