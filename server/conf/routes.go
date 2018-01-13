package conf

import (
	"github.com/gedorinku/koneko-online-judge/server/controllers"
	"github.com/labstack/echo"
)

func Routes(e *echo.Echo) {
	e.POST("/sessions/login", controllers.Login)
	e.DELETE("/sessions/logout", controllers.Logout)

	e.GET("/user", controllers.GetMyUser)
	e.GET("/users", controllers.GetAllUsers)
	e.GET("/users/:name", controllers.GetUser)

	e.POST("/problems/new", controllers.NewProblem)
	e.PUT("/problems/:id", controllers.UpdateProblem)
	e.DELETE("/problems/:id", controllers.DeleteProblem)
	e.GET("/problems", controllers.GetProblems)
	e.GET("/problems/:id", controllers.GetProblem)
	e.POST("/problems/:id/cases/upload", controllers.UpdateCases)
	e.PUT("/problems/:id/cases", controllers.SetTestCasePoint)

	e.GET("/languages", controllers.GetLanguages)
}
