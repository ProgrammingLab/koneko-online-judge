package main

import (
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/gedorinku/koneko-online-judge/server/modules/jobs"
	"github.com/labstack/echo"
)

func main() {
	e := echo.New()
	logger.AppLog = e.Logger
	models.InitDB()
	jobs.InitRunner()
	e.Logger.Fatal(e.Start(":9000"))
}
