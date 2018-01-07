package conf

import (
	"github.com/gedorinku/koneko-online-judge/server/controllers"
	"github.com/labstack/echo"
)

func Routes(e *echo.Echo) {
	e.POST("/sessions/login", controllers.Login)
	e.DELETE("/sessions/logout", controllers.Logout)
}
