package controllers

import (
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models"
	"github.com/labstack/echo"
)

type registrationRequest struct {
	Name        string `json:"name" validate:"required"`
	DisplayName string `json:"displayName" validate:"required"`
	Password    string `json:"password" validate:"required"`
}

type registrationResponse struct {
	User  models.User `json:"user"`
	Token string      `json:"token"`
}

var (
	userNameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_\-.]{3,15}$`)
	displayNameRegex = regexp.MustCompile(`(.){2,25}`)
)

func StartRegistration(c echo.Context) error {
	req := email{}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	email := models.GetWhiteEmail(req.Email)
	if email == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{"Email Not Found"})
	}

	if err := models.StartEmailConfirmation(email); err != nil {
		return ErrInternalServer
	}

	return c.NoContent(http.StatusNoContent)
}

func VerifyEmailConfirmationToken(c echo.Context) error {
	req := c.Param("token")
	token := models.GetEmailConfirmation(req)
	if token == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{"Invalid Token"})
	}

	return c.NoContent(http.StatusNoContent)
}

func RegisterUser(c echo.Context) error {
	p := c.Param("token")
	token := models.GetEmailConfirmation(p)
	if token == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{"Invalid Token"})
	}

	req := registrationRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}
	if !isValidUserName(req.Name) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"user nameは半角英数、'_'、'.'、'-'のみ使用可能で、3文字以上15文字以下である必要があります"})
	}
	if !isValidDisplayName(req.DisplayName) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"display nameは2文字以上25文字以下である必要があります"})
	}
	if !isValidPassword(req.Password) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"パスワードは8文字以上72文字以下で、少なくともアルファベットと数字を含む必要があります"})
	}

	token.FetchWhiteEmail()
	user, err := models.NewUser(req.Name, req.DisplayName, token.WhiteEmail.Email, req.Password, token)
	switch err {
	case models.ErrUserNameAlreadyExists:
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ユーザー名はすでに使われています"})
	case models.ErrEmailAlreadyExists:
		return c.JSON(http.StatusBadRequest, ErrorResponse{"メールアドレスはすでに使われています"})
	case nil:
		_, t, err := models.NewSession(user.Email, req.Password)
		if err != nil {
			logger.AppLog.Errorf("registration error: %+v", err)
			return ErrInternalServer
		}
		resp := registrationResponse{
			User:  *user,
			Token: t,
		}
		return c.JSON(http.StatusCreated, resp)
	default:
		return c.JSON(http.StatusInternalServerError, ErrInternalServer)
	}
}

func isValidUserName(name string) bool {
	return userNameRegex.Match([]byte(name))
}

func isValidDisplayName(displayName string) bool {
	return displayNameRegex.Match([]byte(displayName)) && len(strings.TrimSpace(displayName)) != 0
}

func isValidPassword(password string) bool {
	// bcryptには72bytesより長い文字列はNG🙅‍
	if 72 < len(password) {
		return false
	}

	var (
		count = 0
		alpha = false
		num   = false
	)
	password = strings.ToLower(password)
	for _, c := range password {
		count++
		if unicode.IsLower(c) {
			alpha = true
		}
		if unicode.IsNumber(c) {
			num = true
		}
	}

	return alpha && num && 7 < count
}
