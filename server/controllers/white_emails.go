package controllers

import (
	"net/http"
	"strconv"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo"
)

func GetWhiteEmails(c echo.Context) error {
	if _, err := getAdminSession(c); err != nil {
		return err
	}

	resp := models.GetWhiteEmails()
	for i := range resp {
		resp[i].FetchCreatedBy(false)
	}
	return c.JSON(http.StatusOK, resp)
}

func AddWhiteEmail(c echo.Context) error {
	s, err := getAdminSession(c)
	if err != nil {
		return err
	}
	req := email{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	if exists := models.GetWhiteEmail(req.Email); exists != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"すでに存在するメールアドレスです。"})
	}

	u := models.FindUserByEmail(req.Email)
	if u != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"メールアドレスはすでに使用されています。"})
	}

	e := models.NewWhiteEmail(req.Email, &s.User)
	return c.JSON(http.StatusCreated, e)
}

func DeleteWhiteEmail(c echo.Context) error {
	if _, err := getAdminSession(c); err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.ErrNotFound
	}

	err = models.DeleteWhiteEmail(uint(id))
	switch err {
	case gorm.ErrRecordNotFound:
		return echo.ErrNotFound
	case nil:
		return c.NoContent(http.StatusNoContent)
	default:
		logger.AppLog.Errorf("delete white email error: %+v", err)
		return c.JSON(http.StatusInternalServerError, ErrInternalServer)
	}
}
