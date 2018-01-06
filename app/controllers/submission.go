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

const submissionNotFoundMessage = "submission not found"

func (c Submission) Index(id uint) revel.Result {
	submission := models.GetSubmission(id)
	if submission == nil {
		return c.NotFound(submissionNotFoundMessage)
	}
	submission.FetchUser()
	submission.FetchProblem()
	user := &submission.User
	problem := &submission.Problem
	if !problem.CanView(user) {
		return c.NotFound(submissionNotFoundMessage)
	}
	setResults := submission.GetJudgeSetResultsSorted()
	for i := range setResults {
		setResults[i].FetchCaseSet()
		setResults[i].JudgeResults = setResults[i].GetJudgeResultsSorted()
	}

	initNavigationBar(&c.ViewArgs, c.Session)
	return c.Render(submission, user, problem, setResults, converter)
}

func (c Submission) List(problemID, contestID uint) revel.Result {
	var (
		submissions []models.Submission
		query       string
	)
	user := getUser(c.Session)
	switch {
	case problemID != 0:
		problem := models.GetProblem(problemID)
		if problem == nil || !problem.CanView(user) {
			return c.NotFound(problemNotFoundMessage)
		}
		submissions = problem.GetSubmissionsReversed()
		query = " - " + problem.Title
		problem.FetchSubmissions()
	default:
		return c.NotFound(problemNotFoundMessage)
	}

	for i := range submissions {
		submissions[i].FetchProblem()
		submissions[i].FetchUser()
	}

	initNavigationBar(&c.ViewArgs, c.Session)

	return c.Render(submissions, query, converter)
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
	initNavigationBar(&c.ViewArgs, c.Session)

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
	return c.Redirect(Submission.List, problem.ID, 0)
}
