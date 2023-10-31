package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

// RolesGetUpdate default implementation.
func RolesGetUpdate(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("roles/get_update.html"))
}

func GetRoles(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("roles/get_update.html"))
}

func GetRole(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("roles/get_update.html"))
}

func ListRole(c buffalo.Context) error {
	listRole := &models.MCIamRoles{}
	tx := c.Value("tx").(*pop.Connection)
	resp := handler.ListRole(tx, listRole)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func UpdateRole(c buffalo.Context) error {
	roleId := c.Param("role_id")
	role_bind := &models.MCIamRole{}
	if err := c.Bind(role_bind); err != nil {

	}
	role_bind.ID, _ = uuid.FromString(roleId)
	tx := c.Value("tx").(*pop.Connection)
	handler.UpdateRole(tx, role_bind)

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
