package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

// WorkspaceGetWorkspace default implementation.
func GetWorkspace(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	paramWsId := c.Param("workspaceId")

	resp := handler.GetWorkspace(tx, paramWsId)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func GetWorkspaceList(c buffalo.Context) error {

	tx := c.Value("tx").(*pop.Connection)

	resp := handler.GetWorkspaceList(tx)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func CreateWorkspace(c buffalo.Context) error {
	ws := &models.MCIamWorkspace{}
	err := c.Bind(ws)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]interface{}{
			"error": err,
		}))
	}
	tx := c.Value("tx").(*pop.Connection)
	resp := handler.CreateWorkspace(tx, ws)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func DeleteWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.DeleteWorkspace(tx, paramWsId)
	return c.Render(http.StatusOK, r.JSON(resp))
}
