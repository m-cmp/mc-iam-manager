package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// ProjectGetProject default implementation.
func ProjectGetProject(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("project/get_project.html"))
}
