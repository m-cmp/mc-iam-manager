package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

func GetRole(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	roleId := c.Param("roleId")

	resp := handler.GetRole(tx, roleId)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func ListRole(c buffalo.Context) error {
	listRole := &models.MCIamRoles{}
	tx := c.Value("tx").(*pop.Connection)
	resp := handler.ListRole(tx, listRole)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func UpdateRole(c buffalo.Context) error {
	roleId := c.Param("roleId")
	roleBind := &models.MCIamRole{}
	if err := c.Bind(roleBind); err != nil {
		log.Println("========= role bind error ===========")
		log.Println(err)
		log.Println("========= role bind error ===========")
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}
	roleBind.ID, _ = uuid.FromString(roleId)
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.UpdateRole(tx, roleBind)

	return c.Render(http.StatusOK, r.JSON(resp))
}
func CreateRole(c buffalo.Context) error {
	role_bind := &models.MCIamRole{}
	if err := c.Bind(role_bind); err != nil {
		log.Println("========= role bind error ===========")
		log.Println(err)
		log.Println("========= role bind error ===========")
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}
	log.Println("========= role bind ===========")
	log.Println(role_bind)
	log.Println("========= role bind ===========")

	tx := c.Value("tx").(*pop.Connection)

	resp := handler.CreateRole(tx, role_bind)

	return c.Render(http.StatusAccepted, r.JSON(resp))
}

func DeleteRole(c buffalo.Context) error {
	paramRoleId := c.Param("roleId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.DeleteRole(tx, paramRoleId)
	return c.Render(http.StatusOK, r.JSON(resp))
}
