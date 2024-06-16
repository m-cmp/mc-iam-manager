package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

func CreateWorkspaceUserRoleMappingByName(c buffalo.Context) error {
	m := &models.CreateWorkspaceUserRoleMappingByNameRequest{}
	workspaceName := c.Param("workspaceName")
	err := c.Bind(m)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err.Error()))
	}
	m.WorkspaceName = workspaceName

	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.CreateWorkspaceUserRoleMappingByName(tx, m.WorkspaceName, m.User, m.RoleName)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "there is duplicated mapping..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(workspaceUserRoleMapping))
}

func GetWorkspaceUserRoleMapping(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMapping(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(workspaceUserRoleMapping))
}

func GetWorkspaceUserRoleMappingByWorkspaceName(c buffalo.Context) error {
	workspaceName := c.Param("workspaceName")
	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMappingByWorkspaceName(tx, workspaceName)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(workspaceUserRoleMapping))
}

func GetWorkspaceUserRoleMappingByWorkspacId(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMappingByWorkspaceId(tx, workspaceId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(workspaceUserRoleMapping))
}

func GetWorkspaceUserRoleMappingByUser(c buffalo.Context) error {
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMappingByUser(tx, userId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(workspaceUserRoleMapping))
}

func DeleteWorkspaceUserRoleMappingByName(c buffalo.Context) error {
	workspaceName := c.Param("workspaceName")
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceUserRoleMappingByName(tx, workspaceName, userId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "done"}))
}

func DeleteWorkspaceUserRoleMappingById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceUserRoleMapping(tx, workspaceId, userId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "done"}))
}
