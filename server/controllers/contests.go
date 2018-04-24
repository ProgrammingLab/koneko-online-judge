package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

type contestRequest struct {
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	StartAt      time.Time   `json:"startAt"`
	EndAt        time.Time   `json:"endAt"`
	Writers      []idRequest `json:"writers"`
	Participants []idRequest `json:"participants"`
}

func NewContest(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
	}

	var request contestRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"bind error"})
	}
	if err := c.Validate(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	request.Writers = append(request.Writers, idRequest{s.UserID})
	contest := toContest(&request)
	contest.ID = 0
	if err := models.NewContest(contest); err != nil {
		logger.AppLog.Errorf("new contest error: %v", err.Error())
		return c.JSON(http.StatusInternalServerError, ErrorResponse{"internal server error"})
	}
	return c.JSON(http.StatusCreated, contest)
}

func GetContest(c echo.Context) error {
	s := getSession(c)
	id, err := getContestIDFromContext(c)
	if err != nil {
		return echo.ErrNotFound
	}

	contest := models.GetContestDeeply(uint(id), s)
	if contest == nil {
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, contest)
}

func UpdateContest(c echo.Context) error {
	id, err := getContestIDFromContext(c)
	if err != nil {
		return echo.ErrNotFound
	}

	request := &contestRequest{}
	if err := c.Bind(request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"bind error"})
	}
	if err := c.Validate(request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	request.Participants = nil
	contest := toContest(request)
	contest.ID = uint(id)
	s := getSession(c)
	if !contest.CanEdit(s) {
		return echo.ErrNotFound
	}
	if err := contest.Update(); err != nil {
		logger.AppLog.Error(err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{"internal server error"})
	}
	if len(contest.Writers) != 0 {
		if err := contest.UpdateWriters(); err != nil {
			logger.AppLog.Error(err)
			return c.JSON(http.StatusInternalServerError, ErrorResponse{"internal server error"})
		}
	}

	res := models.GetContestDeeply(contest.ID, s)

	return c.JSON(http.StatusOK, res)
}

func EnterContest(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
	}

	contest := getContestFromContext(c)
	if contest == nil {
		return echo.ErrNotFound
	}

	res, err := contest.IsParticipant(s.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{"internal server error"})
	}
	if res {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"すでに参加しています。"})
	}

	if err := contest.AddParticipant(s.UserID); err != nil {
		logger.AppLog.Error(err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{"internal server error"})
	}

	contest.FetchWriters()
	contest.FetchParticipants()

	return c.JSON(http.StatusOK, contest)
}

func GetStandings(c echo.Context) error {
	s := getSession(c)
	if s == nil {
		return c.JSON(http.StatusUnauthorized, responseUnauthorized)
	}

	contest := getContestFromContext(c)
	if contest == nil || !contest.CanViewProblems(s) {
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, contest.GetStandings())
}

func toContest(request *contestRequest) *models.Contest {
	contest := &models.Contest{
		Title:       request.Title,
		Description: request.Description,
		StartAt:     request.StartAt,
		EndAt:       request.EndAt,
	}

	for _, w := range request.Writers {
		contest.Writers = append(contest.Writers, models.User{ID: w.ID})
	}

	contest.Writers = models.UniqueUsers(contest.Writers)

	return contest
}

func getContestFromContext(c echo.Context) *models.Contest {
	id, err := getContestIDFromContext(c)
	if err != nil {
		return nil
	}

	return models.GetContest(uint(id))
}

func getContestIDFromContext(c echo.Context) (uint, error) {
	id, err := strconv.Atoi(c.Param("contestID"))
	return uint(id), err
}
