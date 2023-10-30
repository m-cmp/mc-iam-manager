package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// RolesGetUpdate default implementation.
func RolesGetUpdate(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("roles/get_update.html"))
}

