package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type createWorkspaceRequset struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func CreateWorkspace(c buffalo.Context) error {
	var req createWorkspaceRequset
	var s models.Workspace
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
	res, err := handler.CreateWorkspace(tx, &s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "workspace is already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(res))
}

func SearchWorkspacesByName(c buffalo.Context) error {
	var err error
	workspaceName := c.Param("workspaceName")
	option := c.Request().URL.Query().Get("option")

	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.SearchWorkspacesByName(tx, workspaceName, option)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetWorkspaceList(c buffalo.Context) error {
	var err error
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetWorkspaceList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetWorkspaceByUUID(c buffalo.Context) error {
	var err error
	workspaceUUID := c.Param("workspaceUUID")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetWorkspaceByUUID(tx, uuid.FromStringOrNil(workspaceUUID))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

type updateWorkspaceByUUIDRequest struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func UpdateWorkspaceByUUID(c buffalo.Context) error {
	var req updateWorkspaceByUUIDRequest
	var err error

	workspaceUUID := c.Param("workspaceUUID")
	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetWorkspaceByUUID(tx, uuid.FromStringOrNil(workspaceUUID))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = handler.CopyStruct(req, s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	res, err := handler.UpdateWorkspace(tx, s)
	if err != nil {
		log.Println(err)
		// err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func DeleteWorkspaceByUUID(c buffalo.Context) error {
	var err error
	workspaceUUID := c.Param("workspaceUUID")
	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetWorkspaceByUUID(tx, uuid.FromStringOrNil(workspaceUUID))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	err = handler.DeleteWorkspace(tx, s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "ID(" + s.ID.String() + ") / Name(" + s.Name + ") is delected.."}))
}
