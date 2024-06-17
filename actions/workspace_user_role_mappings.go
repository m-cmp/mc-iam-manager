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
	WorkspaceID string `json:"workspaceid" `
	UserID      string `json:"userid" `
	RoleID      string `json:"roleid" `
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

func GetWorkspaceUserRoleMappingListByWorkspaceUUID(c buffalo.Context) error {
	workspaceUUID := c.Param("workspaceUUID")
	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.GetWorkspaceUserRoleMappingListByWorkspaceUUID(tx, workspaceUUID)
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

// func UpdateWorkspaceUserRoleMapping(c buffalo.Context) error {
// 	return nil
// }

func DeleteWorkspaceUserRoleMapping(c buffalo.Context) error {
	workspaceUUID := c.Param("workspaceUUID")
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceUserRoleMapping(tx, workspaceUUID, userId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "done"}))
}
