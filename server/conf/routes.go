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

	e.POST("/problems/:id/submissions", controllers.Submit)
	e.GET("/problems/:id/submissions", controllers.GetSubmissions)

	e.GET("/languages", controllers.GetLanguages)

	e.POST("/contests", controllers.NewContest)
	e.GET("/contests/:contestID", controllers.GetContest)
	e.PUT("/contests/:contestID", controllers.UpdateContest)
	e.POST("/contests/:contestID/enter", controllers.EnterContest)
	e.GET("/contests/:contestID/standings", controllers.GetStandings)

	e.POST("/contests/:contestID/problems/new", controllers.NewContestProblem)
	e.GET("/contests/:contestID/problems", controllers.GetContestProblems)

	e.POST("/password_reset", controllers.SendPasswordResetMail)
	e.GET("/password_reset/:token", controllers.VerifyPasswordResetToken)
	e.POST("/password_reset/:token", controllers.ResetPassword)

	e.POST("/white_emails", controllers.AddWhiteEmail)
	e.GET("/white_emails", controllers.GetWhiteEmails)
	e.DELETE("/white_emails/:id", controllers.DeleteWhiteEmail)

	e.POST("/registrations", controllers.StartRegistration)
	e.GET("/registrations/:token", controllers.VerifyEmailConfirmationToken)
}
