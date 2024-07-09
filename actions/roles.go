package actions

import (
	"log"
	"net/http"

	"mc-iam-manager/handler"
	"mc-iam-manager/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type createRoleRequset struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func CreateRole(c buffalo.Context) error {
	var req createRoleRequset
	var s models.Role
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	err = handler.CopyStruct(req, &s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.CreateRole(tx, &s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "Role is already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(res))
}

func SearchRolesByName(c buffalo.Context) error {
	var err error
	roleName := c.Param("roleName")
	option := c.Request().URL.Query().Get("option")

	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.SearchRolesByName(tx, roleName, option)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetRoleList(c buffalo.Context) error {
	var err error
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetRoleList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetRoleById(c buffalo.Context) error {
	var err error
	roleId := c.Param("roleId")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetRoleById(tx, uuid.FromStringOrNil(roleId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

type updateRoleByIdRequest struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func UpdateRoleById(c buffalo.Context) error {
	var req updateRoleByIdRequest
	var err error

	roleId := c.Param("roleId")
	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetRoleById(tx, uuid.FromStringOrNil(roleId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = handler.CopyStruct(req, s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	res, err := handler.UpdateRole(tx, s)
	if err != nil {
		log.Println(err)
		// err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func DeleteRoleById(c buffalo.Context) error {
	var err error
	roleId := c.Param("roleId")
	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetRoleById(tx, uuid.FromStringOrNil(roleId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	err = handler.DeleteRole(tx, s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "ID(" + s.ID.String() + ") / Name(" + s.Name + ") is delected.."}))
}
