package controllers

import (
	"github.com/gedorinku/koneko-online-judge/app/models"
	"github.com/revel/revel"
)

type Submission struct {
	*revel.Controller
}

type SubmissionRequest struct {
	ProblemID  uint
	Language   string
	SourceCode string
}

func (c Submission) SubmitPage(problemID uint) revel.Result {
	problem := models.GetProblem(problemID)
	c.Validation.Required(problem).Message("問題が存在しないか権限がありません 問題ID:%v", problemID)

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(App.Index)
	}

	languages := models.GetAllLanguages()

	return c.Render(problem, languages)
}

func (c Submission) Submit(request *SubmissionRequest) revel.Result {
	c.Validation.Required(request).Message("request is nil")

	c.Validation.
		MaxSize(request.SourceCode, 60000).
		Message("ソースコードはUTF-8で60,000bytes以下である必要があります")

	language := models.GetLanguageByDisplayName(request.Language)
	c.Validation.Required(language).Message("言語は存在しません")

	user := getUser(c.Session)
	problem := models.GetProblem(request.ProblemID)
	if problem == nil || !problem.CanView(user) {
		c.Validation.Error("問題が存在しないか権限がありません%v", problem)
	} else {
		submission := &models.Submission{
			UserID:     user.ID,
			ProblemID:  problem.ID,
			LanguageID: language.ID,
			SourceCode: request.SourceCode,
		}

		err := models.Submit(submission)
		if err != nil {
			c.Validation.Error(err.Error())
		}
	}

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(App.Index)
	}
	return c.Redirect(App.Index)
}
