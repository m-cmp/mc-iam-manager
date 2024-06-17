package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateWorkspaceUserRoleMapping(tx *pop.Connection, s *models.WorkspaceUserRoleMapping) (*models.WorkspaceUserRoleMapping, error) {
	err := tx.Create(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func GetWorkspaceUserRoleMappingListOrderbyWorkspace(tx *pop.Connection) (*[]GetWorkspaceUserRoleMappingListResponse, error) {
	query := "SELECT DISTINCT workspace_id FROM workspace_user_role_mappings;"
	var workspaceIds []string
	err := tx.RawQuery(query).All(&workspaceIds)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var resp []GetWorkspaceUserRoleMappingListResponse
	for _, id := range workspaceIds {
		res, err := GetWorkspaceUserRoleMappingListByWorkspaceUUID(tx, id)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		resp = append(resp, *res)
	}

	return &resp, nil
}

type GetWorkspaceUserRoleMappingListResponse struct {
	Workspace models.Workspace `json:"workspace"`
	UserInfo  []userinfo       `json:"userinfo"`
}

type userinfo struct {
	Userid string      `json:"userid"`
	Role   models.Role `json:"role"`
}

func GetWorkspaceUserRoleMappingListByWorkspaceUUID(tx *pop.Connection, workspaceUUID string) (*GetWorkspaceUserRoleMappingListResponse, error) {
	var resp GetWorkspaceUserRoleMappingListResponse

	ws, err := GetWorkspaceByUUID(tx, uuid.FromStringOrNil(workspaceUUID))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	resp.Workspace = *ws

	var s models.WorkspaceUserRoleMappings
	err = tx.Where("workspace_id = ? ", workspaceUUID).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, wurm := range s {
		var ui userinfo
		ui.Userid = wurm.UserID
		role, err := GetRoleByUUID(tx, wurm.RoleID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		ui.Role = *role
		resp.UserInfo = append(resp.UserInfo, ui)
	}

	return &resp, nil
}

type GetWorkspaceUserRoleMappingListByUserIdResponse struct {
	Role models.Role                          `json:"role"`
	WPM  getWPmappingListOrderbyWorkspaceResp `json:"workspaceProject"`
}

func GetWorkspaceUserRoleMappingListByUserId(tx *pop.Connection, userId string) (*[]GetWorkspaceUserRoleMappingListByUserIdResponse, error) {
	var s models.WorkspaceUserRoleMappings
	err := tx.Where("user_id = ? ", userId).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var resp []GetWorkspaceUserRoleMappingListByUserIdResponse
	for _, i := range s {
		var res GetWorkspaceUserRoleMappingListByUserIdResponse
		role, err := GetRoleByUUID(tx, i.RoleID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		res.Role = *role
		wpm, err := GetWPmappingListByWorkspaceUUID(tx, i.WorkspaceID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		res.WPM = *wpm
		resp = append(resp, res)
	}

	return &resp, nil
}

func DeleteWorkspaceUserRoleMapping(tx *pop.Connection, workspaceUUID string, userId string) error {
	var s models.WorkspaceUserRoleMapping
	err := tx.Where("workspace_id = ? and user_id = ?", workspaceUUID, userId).First(&s)
	if err != nil {
		log.Error(err)
		return err
	}
	err = tx.Destroy(&s)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
