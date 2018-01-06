package controllers

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/app/models"
	"github.com/jinzhu/gorm"
	"github.com/revel/revel"
)

type Contest struct {
	*revel.Controller
}

type ContestRequest struct {
	ID          uint
	Title       string
	Description string
	Writers     []string
	StartAt     string
	EndAt       string
}

const contestNotFoundMessage = "コンテストが存在しないか、権限がありません。"

func (c Contest) Edit(id uint) revel.Result {
	user := getUser(c.Session)
	var contest *models.Contest
	if id == 0 {
		contest = models.GetDefaultContest(user)
	} else {
		contest = models.GetContest(id)
	}
	if contest == nil {
		return c.NotFound(contestNotFoundMessage)
	}

	initNavigationBar(&c.ViewArgs, c.Session)

	return c.Render(contest, converter)
}

func (c Contest) Update(request ContestRequest) revel.Result {
	contest, err := request.toContest()
	if err != nil {
		c.Validation.Error("日時をパースできません")
	}

	err = contest.Update()
	if err != nil {
		revel.AppLog.Error("error", err)
		c.Validation.Error("保存に失敗しました")
	}

	return c.Redirect(Contest.Edit, contest.ID)
}

func (r ContestRequest) toContest() (*models.Contest, error) {
	startAt, err := time.Parse(htmlDateTimeLayout, r.StartAt)
	if err != nil {
		revel.AppLog.Error("", err)
		return nil, err
	}
	endAt, err := time.Parse(htmlDateTimeLayout, r.EndAt)
	if err != nil {
		revel.AppLog.Error("", err)
		return nil, err
	}
	contest := &models.Contest{
		Model: gorm.Model{
			ID: r.ID,
		},
		Title:       r.Title,
		Description: r.Description,
		StartAt:     startAt,
		EndAt:       endAt,
	}

	return contest, nil
}

func toContestRequest(contest *models.Contest) *ContestRequest {
	contest.FetchWriters()
	writers := make([]string, len(contest.Writers))
	for i, w := range contest.Writers {
		writers[i] = w.Name
	}
	request := &ContestRequest{
		ID:          contest.ID,
		Title:       contest.Title,
		Description: contest.Description,
		Writers:     writers,
		StartAt:     contest.StartAt.Format(htmlDateTimeLayout),
		EndAt:       contest.EndAt.Format(htmlDateTimeLayout),
	}

	return request
}
