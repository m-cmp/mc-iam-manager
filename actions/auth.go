package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func LoginWithPasswordHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "LoginWithPasswordHandler"}))
}

func LogoutHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "LogoutHandler"}))
}
