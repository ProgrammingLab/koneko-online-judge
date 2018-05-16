package controllers

import (
	"github.com/labstack/echo"
)

func Routes(e *echo.Echo) {
	e.POST("/sessions/login", Login)
	e.DELETE("/sessions/logout", Logout)

	e.GET("/user", GetMyUser)
	e.GET("/users", GetAllUsers)
	e.GET("/users/:name", GetUser)

	e.POST("/problems/new", NewProblem)
	e.PUT("/problems/:id", UpdateProblem)
	e.DELETE("/problems/:id", DeleteProblem)
	e.GET("/problems", GetProblems)
	e.GET("/problems/:id", GetProblem)
	e.POST("/problems/:id/cases/upload", UpdateCases)
	e.PUT("/problems/:id/cases", SetTestCasePoint)
	e.POST("/problems/:id/rejudge", RejudgeProblem)

	e.POST("/problems/:id/submissions", Submit)
	e.GET("/problems/:id/submissions", GetSubmissions)

	e.POST("/submissions/:id/rejudge", Rejudge)
	e.GET("/submissions/:id", GetSubmission)

	e.GET("/languages", GetLanguages)

	e.POST("/contests", NewContest)
	e.GET("/contests", GetContests)
	e.GET("/contests/:contestID", GetContest)
	e.PUT("/contests/:contestID", UpdateContest)
	e.POST("/contests/:contestID/enter", EnterContest)
	e.GET("/contests/:contestID/standings", GetStandings)
	e.GET("/contests/:contestID/submissions", GetContestSubmissions)
	e.GET("/contests/:contestID/statuses", GetContestJudgeStatuses)

	e.POST("/contests/:contestID/problems/new", NewContestProblem)
	e.GET("/contests/:contestID/problems", GetContestProblems)

	e.POST("/password_reset", SendPasswordResetMail)
	e.GET("/password_reset/:token", VerifyPasswordResetToken)
	e.POST("/password_reset/:token", ResetPassword)

	e.POST("/white_emails", AddWhiteEmail)
	e.GET("/white_emails", GetWhiteEmails)
	e.DELETE("/white_emails/:id", DeleteWhiteEmail)

	e.POST("/registrations", StartRegistration)
	e.GET("/registrations/:token", VerifyEmailConfirmationToken)
	e.POST("/registrations/:token", RegisterUser)

	e.GET("/workers", GetWorkerStatus)
}
