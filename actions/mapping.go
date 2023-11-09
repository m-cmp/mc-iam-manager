package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/pkg/errors"
)

func MappingWsUser(c buffalo.Context) error {
	wum := &models.MCIamWsUserMapping{}
	if err := c.Bind(wum); err != nil {

	}
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.MappingWsUser(tx, wum)
	return c.Render(http.StatusOK, r.JSON(resp))
}

// MappingWsUserRoleMapping default implementation.
func MappingWsUserRole(c buffalo.Context) error {
	wurm := &models.MCIamWsUserRoleMapping{}
	if err := c.Bind(wurm); err != nil {

	}
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.MappingWsUserRole(tx, wurm)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingUserRole(c buffalo.Context) error {
	urm := &models.MCIamUserRoleMapping{}

	if err := c.Bind(urm); err != nil {

	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingUserRole(tx, urm)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingWsProject(c buffalo.Context) error {
	wp := &models.MCIamWsProjectMapping{}

	if err := c.Bind(wp); err != nil {

	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingWsProject(tx, wp)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingGetProjectByWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingGetProjectByWorkspace(tx, paramWsId)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingDeleteWsProject(c buffalo.Context) error {
	// paramWsId := c.Param("workspaceId")
	// paramProjectId := c.Param("projectId")

	bindModel := &models.MCIamWsProjectMapping{}

	if err := c.Bind(bindModel); err != nil {
		return errors.WithStack(err)
	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingDeleteWsProject(tx, bindModel)

	return c.Render(http.StatusOK, r.JSON(resp))

}
