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

type createWorkspaceUserRoleMappingRequest struct {
	WorkspaceID string `json:"workspaceId" `
	UserID      string `json:"userId" `
	RoleID      string `json:"roleId" `
}

func CreateWorkspaceUserRoleMapping(c buffalo.Context) error {
	var req createWorkspaceUserRoleMappingRequest
	var s models.WorkspaceUserRoleMapping
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	s.WorkspaceID = uuid.FromStringOrNil(req.WorkspaceID)
	s.UserID = req.UserID
	s.RoleID = uuid.FromStringOrNil(req.RoleID)

	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.CreateWorkspaceUserRoleMapping(tx, &s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(res))
}

func GetWorkspaceUserRoleMappingListOrderbyWorkspace(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.GetWorkspaceUserRoleMappingListOrderbyWorkspace(tx)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(resp))
}

func GetWorkspaceUserRoleMappingListByWorkspaceId(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.GetWorkspaceUserRoleMappingListByWorkspaceId(tx, workspaceId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(resp))
}

func GetWorkspaceUserRoleMappingListByUserId(c buffalo.Context) error {
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.GetWorkspaceUserRoleMappingListByUserId(tx, userId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(resp))
}

func GetWorkspaceUserRoleMappingById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.GetWorkspaceUserRoleMappingById(tx, workspaceId, userId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(resp))
}

func DeleteWorkspaceUserRoleMapping(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceUserRoleMapping(tx, workspaceId, userId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "done"}))
}
