package handler

import (
	"github.com/m-cmp/mc-iam-manager/models"

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
		res, err := GetWorkspaceUserRoleMappingListByWorkspaceId(tx, id)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		resp = append(resp, *res)
	}

	return &resp, nil
}

func GetWorkspaceUserRoleMappingById(tx *pop.Connection, workspaceId string, userId string) (*models.Role, error) {
	var s models.WorkspaceUserRoleMapping
	err := tx.Where("workspace_id = ? and user_id = ?", workspaceId, userId).First(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	role, err := GetRoleById(tx, s.RoleID)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return role, nil
}

type GetWorkspaceUserRoleMappingListResponse struct {
	Workspace models.Workspace `json:"workspace"`
	UserInfo  []userinfo       `json:"userinfo"`
}

type userinfo struct {
	Userid string      `json:"userid"`
	Role   models.Role `json:"role"`
}

func GetWorkspaceUserRoleMappingListByWorkspaceId(tx *pop.Connection, workspaceId string) (*GetWorkspaceUserRoleMappingListResponse, error) {
	var resp GetWorkspaceUserRoleMappingListResponse

	ws, err := GetWorkspaceById(tx, uuid.FromStringOrNil(workspaceId))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	resp.Workspace = *ws

	var s models.WorkspaceUserRoleMappings
	err = tx.Where("workspace_id = ? ", workspaceId).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, wurm := range s {
		var ui userinfo
		ui.Userid = wurm.UserID
		role, err := GetRoleById(tx, wurm.RoleID)
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
		role, err := GetRoleById(tx, i.RoleID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		res.Role = *role
		wpm, err := GetWPmappingListByWorkspaceId(tx, i.WorkspaceID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		res.WPM = *wpm
		resp = append(resp, res)
	}

	return &resp, nil
}

func DeleteWorkspaceUserRoleMapping(tx *pop.Connection, workspaceId string, userId string) error {
	var s models.WorkspaceUserRoleMapping
	err := tx.Where("workspace_id = ? and user_id = ?", workspaceId, userId).First(&s)
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
