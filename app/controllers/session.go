package controllers

import (
	"github.com/revel/revel"
)

type Session struct {
	*revel.Controller
}

func (c Session) Login() revel.Result {
	return c.Render()
}
