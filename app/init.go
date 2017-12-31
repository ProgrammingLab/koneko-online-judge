package app

import (
	"strconv"
	"github.com/revel/revel"

	"github.com/gedorinku/koneko-online-judge/app/models"
	"github.com/gedorinku/koneko-online-judge/app/controllers"
)

var (
	// AppVersion revel app version (ldflags)
	AppVersion string

	// BuildTime revel app build-time (ldflags)
	BuildTime string
)

func init() {
	// Filters is the default set of global filters.
	revel.Filters = []revel.Filter{
		revel.PanicFilter,             // Recover from panics and display an error page instead.
		revel.RouterFilter,            // Use the routing table to select the right Action
		revel.FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		revel.ParamsFilter,            // Parse parameters into Controller.Params.
		revel.SessionFilter,           // Restore and write the session cookie.
		revel.FlashFilter,             // Restore and write the flash cookie.
		revel.ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
		revel.I18nFilter,              // Resolve the requested language
		HeaderFilter,                  // Add some security based headers
		revel.InterceptorFilter,       // Run interceptors around the action.
		revel.CompressFilter,          // Compress the result.
		revel.ActionInvoker,           // Invoke the action.
	}

	revel.OnAppStart(models.InitDB)

	revel.InterceptFunc(checkLogin, revel.BEFORE, &controllers.App{})
	revel.InterceptFunc(checkLogin, revel.BEFORE, &controllers.Problem{})
	revel.InterceptFunc(checkLogin, revel.BEFORE, &controllers.Submission{})
}

// HeaderFilter adds common security headers
// There is a full implementation of a CSRF filter in
// https://github.com/revel/modules/tree/master/csrf
var HeaderFilter = func(c *revel.Controller, fc []revel.Filter) {
	c.Response.Out.Header().Add("X-Frame-Options", "SAMEORIGIN")
	c.Response.Out.Header().Add("X-XSS-Protection", "1; mode=block")
	c.Response.Out.Header().Add("X-Content-Type-Options", "nosniff")

	fc[0](c, fc[1:]) // Execute the next filter stage.
}

func checkLogin(c *revel.Controller) revel.Result {
	if c.Request.GetPath() == "/login" {
		return nil
	}
	userID, _ := strconv.Atoi(c.Session[controllers.SessionUserIDKey])
	token := c.Session[controllers.SessionTokenKey]
	if session := models.CheckLogin(uint(userID), token); session == nil {
		c.Flash.Error("ログインしてください")
		return c.Redirect(controllers.App.LoginPage)
	}

	return nil
}
