package main

import (
	"github.com/gedorinku/koneko-online-judge/server/conf"
	"github.com/gedorinku/koneko-online-judge/server/controllers"
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/go-playground/validator.v9"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	logger.AppLog = e.Logger
	models.InitDB()

	e.Validator = &CustomValidator{validator: validator.New()}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(controllers.CheckLogin)

	conf.Routes(e)

	models.InitJobs()
	defer models.StopPool()

	e.Logger.Fatal(e.Start(":9000"))
}
