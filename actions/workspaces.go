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

func GetWorkspaceById(c buffalo.Context) error {
	var err error
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetWorkspaceById(tx, uuid.FromStringOrNil(workspaceId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

type updateWorkspaceByIdRequest struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func UpdateWorkspaceById(c buffalo.Context) error {
	var req updateWorkspaceByIdRequest
	var err error

	workspaceId := c.Param("workspaceId")
	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetWorkspaceById(tx, uuid.FromStringOrNil(workspaceId))
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

func DeleteWorkspaceById(c buffalo.Context) error {
	var err error
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetWorkspaceById(tx, uuid.FromStringOrNil(workspaceId))
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
