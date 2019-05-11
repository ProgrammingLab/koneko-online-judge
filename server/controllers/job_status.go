package controllers

import (
	"net/http"

	"github.com/ProgrammingLab/koneko-online-judge/server/models"
	"github.com/gocraft/work"
	"github.com/labstack/echo"
)

type scheduledJobsResponse struct {
	Total         int64                `json:"total"`
	ScheduledJobs []*work.ScheduledJob `json:"scheduled_jobs"`
}

func GetWorkerStatus(c echo.Context) error {
	s, err := getAdminSession(c)
	if err != nil {
		return ErrInternalServer
	}
	if s == nil {
		return echo.ErrNotFound
	}

	q, err := models.GetWorkers()
	if err != nil {
		return ErrInternalServer
	}

	return c.JSON(http.StatusOK, q)
}
