package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

// RolesGetUpdate default implementation.
func RolesGetUpdate(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("roles/get_update.html"))
}

func CreateRole(c buffalo.Context) error {
	role_bind := &models.MCIamRole{}
	if err := c.Bind(role_bind); err != nil {

	}
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.CreateRole(tx, role_bind)

	return c.Render(http.StatusAccepted, r.JSON(resp))
}
