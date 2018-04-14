package controllers

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

func NewProblem(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
	}

	problem := &models.Problem{}
	if err := c.Bind(problem); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(problem); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if problem.JudgeType != models.JudgeTypeNormal && problem.JudgementConfig == nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"judgementConfig is required"})
	}

	s.FetchUser()
	problem.ID = 0
	contestID, err := getContestIDFromContext(c)
	if err == nil {
		problem.ContestID = &contestID
	} else {
		problem.ContestID = nil
	}
	problem.Contest = nil
	problem.WriterID = s.ID
	problem.Writer = s.User
	if err := models.NewProblem(problem); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
	}
	if problem.JudgementConfig != nil {
		problem.JudgementConfig.FetchLanguage()
	}
	if problem.ContestID == nil {
		problem.ContestID = new(uint)
	}
	return c.JSON(http.StatusCreated, problem)
}

func UpdateProblem(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
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

	problem := getProblemFromContext(c)
	if problem == nil || !problem.CanEdit(s) {
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

	s := getSession(c)

	problems := models.GetProblems(nil, uint(minID), uint(maxID), count)
	results := make([]models.Problem, 0, len(problems))
	for i := range problems {
		if !problems[i].CanView(s) {
			continue
		}
		fetchProblem(&problems[i], s)
		results = append(results, problems[i])
	}

	return c.JSON(http.StatusOK, results)
}

func GetProblem(c echo.Context) error {
	s := getSession(c)
	problem := getProblemFromContext(c)
	if problem == nil || !problem.CanView(s) {
		return echo.ErrNotFound
	}

	fetchProblem(problem, s)

	return c.JSON(http.StatusOK, problem)
}

func DeleteProblem(c echo.Context) error {
	s := getSession(c)
	problem := getProblemFromContext(c)
	if problem == nil || !problem.CanEdit(s) {
		return echo.ErrNotFound
	}

	problem.Delete()

	return c.NoContent(http.StatusNoContent)
}

func UpdateCases(c echo.Context) error {
	s := getSession(c)
	problem := getProblemFromContext(c)
	if problem == nil || !problem.CanEdit(s) {
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

func SetTestCasePoint(c echo.Context) error {
	s := getSession(c)
	problem := getProblemFromContext(c)
	if problem == nil || !problem.CanEdit(s) {
		return echo.ErrNotFound
	}

	requests := make([]int, 0)
	if err := c.Bind(&requests); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	for _, r := range requests {
		if r < 0 {
			return c.JSON(http.StatusBadRequest, ErrorResponse{"点数は0以上である必要があります"})
		}
	}

	problem.FetchCaseSets()
	for i, s := range problem.CaseSets {
		s.UpdatePoint(requests[i])
	}

	return c.NoContent(http.StatusNoContent)
}

func RejudgeProblem(c echo.Context) error {
	_, err := getAdminSession(c)
	if err != nil {
		return err
	}

	p := getProblemFromContext(c)
	if p == nil {
		return echo.ErrNotFound
	}

	if err := p.Rejudge(); err != nil {
		return ErrInternalServer
	}

	return c.NoContent(http.StatusNoContent)
}

func fetchProblem(out *models.Problem, s *models.UserSession) {
	out.FetchWriter()
	out.FetchSamples()
	out.FetchCaseSets()
	out.FetchContest()
	if out.CanEdit(s) {
		out.FetchJudgementConfig()
	}

	out.Writer.Email = ""
	if out.ContestID == nil {
		out.ContestID = new(uint)
	}

	if out.JudgementConfigID == nil {
		out.JudgementConfigID = new(uint)
	}
}

func getProblemFromContext(c echo.Context) *models.Problem {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return nil
	}

	return models.GetProblem(uint(id))
}
