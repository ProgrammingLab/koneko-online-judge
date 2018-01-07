package main

import (
	"github.com/gedorinku/koneko-online-judge/server/conf"
	"github.com/gedorinku/koneko-online-judge/server/controllers"
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/gedorinku/koneko-online-judge/server/modules/jobs"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
)

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	logger.AppLog = e.Logger
	models.InitDB()
	jobs.InitRunner()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(controllers.CheckLogin)

	conf.Routes(e)

	e.Logger.Fatal(e.Start(":9000"))
}
