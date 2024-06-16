package actions

import (
	"fmt"
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"

	"github.com/gobuffalo/buffalo"
)

type createWorkspaceRequset struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func CreateWorkspace(c buffalo.Context) error {
	workspace := &models.Workspace{}
	workspaceReq := &createWorkspaceRequset{}

	err := c.Bind(workspaceReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*workspaceReq, workspace)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	createdWorkspace, err := handler.CreateWorkspace(tx, workspace)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "workspace name is duplicated...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(createdWorkspace))
}

func GetWorkspaceList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	workspaceList, err := handler.GetWorkspaceList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}
	if len(*workspaceList) == 0 {
		return c.Render(http.StatusOK, r.JSON([]map[string]string{}))
	}
	return c.Render(http.StatusOK, r.JSON(workspaceList))
}

func GetWorkspaceByName(c buffalo.Context) error {
	workspaceName := c.Param("workspaceName")
	tx := c.Value("tx").(*pop.Connection)
	workspace, err := handler.GetWorkspaceByName(tx, workspaceName)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(workspace))
}

func GetWorkspaceById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	workspace, err := handler.GetWorkspaceById(tx, workspaceId)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(workspace))
}

func UpdateWorkspaceByName(c buffalo.Context) error {
	workspaceName := c.Param("workspaceName")

	workspace := &models.Workspace{}
	workspaceReq := &createWorkspaceRequset{}

	err := c.Bind(workspaceReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*workspaceReq, workspace)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	updatedworkspace, err := handler.UpdateWorkspaceByname(tx, workspaceName, workspace)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "the workspace you are trying to change is duplicated...")
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the workspace you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(updatedworkspace))
}

func UpdateWorkspaceById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	workspace := &models.Workspace{}
	workspaceReq := &createWorkspaceRequset{}

	err := c.Bind(workspaceReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*workspaceReq, workspace)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	updatedworkspace, err := handler.UpdateWorkspaceById(tx, workspaceId, workspace)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "the workspace you are trying to change is duplicated...")
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the workspace you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(updatedworkspace))
}

func DeleteWorkspaceByName(c buffalo.Context) error {
	workspaceName := c.Param("workspaceName")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceByName(tx, workspaceName)
	fmt.Println("DeleteWorkspaceByName handler done")
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no workspace ("+workspaceName+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": workspaceName + " is deleted..."}))
}

func DeleteWorkspaceById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceById(tx, workspaceId)
	fmt.Println("DeleteWorkspaceByName handler done")
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no workspace ("+workspaceId+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": workspaceId + " is deleted..."}))
}
