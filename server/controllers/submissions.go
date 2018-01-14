package controllers

import (
	"net/http"

	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

type submissionRequest struct {
	Language   string `json:"language"`
	SourceCode string `json:"sourceCode"`
}

func Submit(c echo.Context) error {
	s := getSession(c)
	problem := getProblemFromContext(c)
	if problem == nil || !problem.CanView(s.UserID) {
		return echo.ErrNotFound
	}

	request := &submissionRequest{}
	if err := c.Bind(request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	lang := models.GetLanguageByDisplayName(request.Language)
	if lang == nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"使用できない言語です"})
	}

	submission := &models.Submission{
		UserID:     s.UserID,
		ProblemID:  problem.ID,
		LanguageID: lang.ID,
		SourceCode: request.SourceCode,
	}

	if err := models.Submit(submission); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
	}

	fetchSubmission(submission, s.UserID)

	return c.JSON(http.StatusCreated, submission)
}

func fetchSubmission(out *models.Submission, userID uint) {
	out.FetchUser()
	out.User.Email = ""
	fetchProblem(&out.Problem, userID)
	out.Problem.ContestID = new(uint)
	out.FetchJudgeSetResultsDeeply(true)
	out.FetchLanguage()
}
