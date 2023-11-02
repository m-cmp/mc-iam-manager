package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

// MappingWsUserRoleMapping default implementation.
func MappingWsUserRoleMapping(c buffalo.Context) error {
	wurm := &models.MCIamWsUserRoleMapping{}
	if err := c.Bind(wurm); err != nil {

	}
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.WsUserRoleMapping(tx, wurm)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func UserRoleMapping(c buffalo.Context) error {
	urm := &models.MCIamUserRoleMapping{}

	if err := c.Bind(urm); err != nil {

	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.UserRoleMapping(tx, urm)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func WorkspaceProjectMapping(c buffalo.Context) error {
	wp := &models.MCIamWsProjectMapping{}

	if err := c.Bind(wp); err != nil {

	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.WsProjectMapping(tx, wp)

	return c.Render(http.StatusOK, r.JSON(resp))
}
