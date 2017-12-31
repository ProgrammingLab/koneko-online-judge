package controllers

import (
	"unicode/utf8"
	"time"
	"github.com/revel/revel"
	"github.com/gedorinku/koneko-online-judge/app/models"
)

type Problem struct {
	*revel.Controller
}

type ProblemRequest struct {
	Title            string
	TimeLimitSeconds int
	MemoryLimit      int
	Body             string
}

var (
	problemNotFoundMessage      = "problem not found: id %v"
	problemEditForbiddenMessage = "editing is not allowed"
)

func (c Problem) New() revel.Result {
	user := getUser(c.Session)
	problem := models.NewProblem(user)
	return c.Redirect(Problem.Edit, problem.ID)
}

func (c Problem) Edit(id uint) revel.Result {
	problem := models.GetProblem(id)
	if problem == nil {
		return c.NotFound(problemNotFoundMessage, id)
	}

	user := getProblemEditUser(c.Session, problem)
	if user == nil {
		return c.Forbidden(problemEditForbiddenMessage)
	}

	initNavigationBar(&c.ViewArgs, c.Session)
	return c.Render(problem)
}

func (c Problem) Update(id uint, request *ProblemRequest, caseArchive []byte) revel.Result {
	problem := models.GetProblem(id)
	if problem == nil {
		return c.NotFound(problemNotFoundMessage)
	}

	user := getProblemEditUser(c.Session, problem)
	if user == nil {
		return c.Forbidden(problemEditForbiddenMessage)
	}

	c.Validation.
		Required(request).
		Message("problemがnilです。")
	c.Validation.
		Range(utf8.RuneCountInString(request.Title), 1, 40).
		Message("タイトルは1文字以上40文字以下である必要があります。")
	c.Validation.
		Range(request.TimeLimitSeconds, 1, 60).
		Message("時間制限は1秒以上60秒以下である必要があります。")
	c.Validation.
		Range(request.MemoryLimit, 128, 512).
		Message("メモリ制限は128MiB以上512MiB以下である必要があります。")
	c.Validation.
		Max(len(caseArchive), 1024*1024*10).
		Message("テストケースは10MiB以下である必要があります。")
	c.Validation.
		Max(len(request.Body), 60000).
		Message("問題文はUTF-8で60,000bytes以下である必要があります。")

	tmp := &models.Problem{
		Title:       request.Title,
		TimeLimit:   time.Duration(request.TimeLimitSeconds) * time.Second,
		MemoryLimit: request.MemoryLimit,
		Body:        request.Body,
	}

	problem.Update(tmp)

	if len(caseArchive) != 0 {
		err := problem.ReplaceTestCases(caseArchive[:])
		if err != nil {
			c.Validation.Error(err.Error())
		}
	}

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Problem.Edit, id)
	}
	c.Flash.Success("保存しました。")
	c.FlashParams()
	return c.Redirect(Problem.Edit, id)
}

func getProblemEditUser(session revel.Session, problem *models.Problem) *models.User {
	user := getUser(session)
	if user.ID != problem.WriterID {
		return nil
	}
	return user
}
