package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// WorkspaceGetWorkspace default implementation.
func WorkspaceGetWorkspace(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("workspace/get_workspace.html"))
}

func CreateWorkspace(c buffalo.Context) error {
	ws := &models.MCIamWorkspace{}
	err := c.Bind(ws)
	if err != nil {

	}
	resp := handler.CreateWorkspace(tx, ws)
	return c.Render(http.StatusOK, r.JSON(resp))
}
