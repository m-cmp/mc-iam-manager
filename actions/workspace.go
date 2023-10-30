package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

// WorkspaceGetWorkspace default implementation.
func GetWorkspace(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("workspace/get_workspace.html"))
}

func CreateWorkspace(c buffalo.Context) error {
	ws := &models.MCIamWorkspace{}
	err := c.Bind(ws)
	if err != nil {

	}
	tx := c.Value("tx").(*pop.Connection)
	resp := handler.CreateWorkspace(tx, ws)
	return c.Render(http.StatusOK, r.JSON(resp))
}
