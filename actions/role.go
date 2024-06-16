package actions

import (
	"fmt"
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"

	"github.com/gobuffalo/buffalo"
)

type createRoleRequset struct {
	Name    string `json:"name" db:"name"`
	Idp     string `json:"idp" db:"idp"`
	IdpUUID string `json:"idp_uuid" db:"idp_uuid"`
}

func CreateRole(c buffalo.Context) error {
	Role := &models.Role{}
	RoleReq := &createRoleRequset{}

	err := c.Bind(RoleReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*RoleReq, Role)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	createdRole, err := handler.CreateRole(tx, Role)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "Role name is duplicated...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(createdRole))
}

func GetRoleList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	RoleList, err := handler.GetRoleList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}
	if len(*RoleList) == 0 {
		return c.Render(http.StatusOK, r.JSON([]map[string]string{}))
	}
	return c.Render(http.StatusOK, r.JSON(RoleList))
}

func GetRoleByName(c buffalo.Context) error {
	RoleName := c.Param("roleName")
	tx := c.Value("tx").(*pop.Connection)
	Role, err := handler.GetRoleByName(tx, RoleName)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(Role))
}

func GetRoleById(c buffalo.Context) error {
	RoleId := c.Param("roleId")
	tx := c.Value("tx").(*pop.Connection)
	Role, err := handler.GetRoleById(tx, RoleId)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(Role))
}

func UpdateRoleByName(c buffalo.Context) error {
	RoleName := c.Param("roleName")

	Role := &models.Role{}
	RoleReq := &createRoleRequset{}

	err := c.Bind(RoleReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*RoleReq, Role)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	updatedRole, err := handler.UpdateRoleByname(tx, RoleName, Role)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "the Role you are trying to change is duplicated...")
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the Role you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(updatedRole))
}

func UpdateRoleById(c buffalo.Context) error {
	RoleId := c.Param("roleId")

	Role := &models.Role{}
	RoleReq := &createRoleRequset{}

	err := c.Bind(RoleReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*RoleReq, Role)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	updatedRole, err := handler.UpdateRoleById(tx, RoleId, Role)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "the Role you are trying to change is duplicated...")
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the Role you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(updatedRole))
}

func DeleteRoleByName(c buffalo.Context) error {
	RoleName := c.Param("roleName")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteRoleByName(tx, RoleName)
	fmt.Println("DeleteRoleByName handler done")
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no Role ("+RoleName+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": RoleName + " is deleted..."}))
}

func DeleteRoleById(c buffalo.Context) error {
	RoleId := c.Param("roleId")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteRoleById(tx, RoleId)
	fmt.Println("DeleteRoleByName handler done")
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no Role ("+RoleId+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": RoleId + " is deleted..."}))
}
