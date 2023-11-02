package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

// ProjectGetProject default implementation.
func GetProject(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	paramProjectId := c.Param("projectId")

	resp := handler.GetProject(tx, paramProjectId)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func GetProjectList(c buffalo.Context) error {

	tx := c.Value("tx").(*pop.Connection)

	resp := handler.GetProjectList(tx)
	return c.Render(http.StatusOK, r.JSON(resp))
}
func CreateProject(c buffalo.Context) error {
	pj := &models.MCIamProject{}
	err := c.Bind(pj)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]interface{}{
			"error": err,
		}))
	}
	tx := c.Value("tx").(*pop.Connection)
	resp := handler.CreateProject(tx, pj)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func DeleteProject(c buffalo.Context) error {
	paramPjId := c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.DeleteProject(tx, paramPjId)
	return c.Render(http.StatusOK, r.JSON(resp))
}
