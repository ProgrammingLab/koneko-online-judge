package controllers

import (
	"github.com/revel/revel"
	"github.com/gedorinku/koneko-online-judge/app/models"
)

type Problem struct {
	*revel.Controller
}

func (c Problem) New() revel.Result {
	user := getUser(c.Session)
	problem := models.NewProblem(user)
	return c.Redirect(Problem.Edit, problem.ID)
}

func (c Problem) Edit(id uint) revel.Result {
	initNavigationBar(&c.ViewArgs, c.Session)
	return c.Render()
}
