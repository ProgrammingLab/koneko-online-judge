package controllers

import (
	"net/http"

	"github.com/labstack/echo"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

var ErrInternalServer = echo.NewHTTPError(http.StatusInternalServerError)
