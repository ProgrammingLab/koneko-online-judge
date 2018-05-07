package controllers

import "github.com/labstack/echo"

func AddAccessControlAllowHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Access-Control-Allow-Headers", "Date")
		return next(c)
	}
}
