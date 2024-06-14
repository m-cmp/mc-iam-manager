package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"

	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

func CreateRole(c buffalo.Context) error {
	role := &models.MCIamRoletype{}
	err := c.Bind(role)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	role.Type = role.RoleName // TODO : 모두 필요한가?
	role.RoleID = role.RoleName

	tx := c.Value("tx").(*pop.Connection)
	createdRole, err := handler.CreateRole(tx, role)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, createdRole)),
	)
}

func GetRoleList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	roleList, err := handler.GetRoleList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, roleList)),
	)
}

func GetRole(c buffalo.Context) error {
	roleId := c.Param("roleId")

	tx := c.Value("tx").(*pop.Connection)
	roleList, err := handler.GetRole(tx, roleId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, roleList)),
	)
}

func UpdateRole(c buffalo.Context) error {
	role := &models.MCIamRoletype{}
	err := c.Bind(role)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	role.RoleID = c.Param("roleId")

	tx := c.Value("tx").(*pop.Connection)
	RoleList, err := handler.UpdateRole(tx, role)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, RoleList)),
	)
}

func DeleteRole(c buffalo.Context) error {
	RoleId := c.Param("roleId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteRole(tx, RoleId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}
	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, nil)),
	)
}
