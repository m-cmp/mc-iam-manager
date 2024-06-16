package handler

import (
	"fmt"
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateWorkspaceUserRoleMapping(tx *pop.Connection, m *models.MappingWorkspaceUserRole) (*models.MappingWorkspaceUserRole, error) {
	err := tx.Create(m)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return m, nil
}

func CreateWorkspaceUserRoleMappingByName(tx *pop.Connection, workspaceName string, userId string, roleName string) (*models.MappingWorkspaceUserRole, error) {
	worksapceId, err := GetWorkspaceIdByname(tx, workspaceName)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	roleId, err := GetRoleIdByname(tx, roleName)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	m := &models.MappingWorkspaceUserRole{
		WorkspaceID: worksapceId,
		User:        userId,
		RoleID:      roleId,
	}

	workspaceUserRoleMapping, err := CreateWorkspaceUserRoleMapping(tx, m)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return workspaceUserRoleMapping, nil
}

func CreateWorkspaceUserRoleMappingById(tx *pop.Connection, workspaceId string, userId string, roleId string) (*models.MappingWorkspaceUserRole, error) {
	m := &models.MappingWorkspaceUserRole{
		WorkspaceID: uuid.FromStringOrNil(workspaceId),
		User:        userId,
		RoleID:      uuid.FromStringOrNil(roleId),
	}

	workspaceUserRoleMapping, err := CreateWorkspaceUserRoleMapping(tx, m)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return workspaceUserRoleMapping, nil
}

func GetWorkspaceUserRoleMapping(tx *pop.Connection) (*models.MappingWorkspaceUserRoleResponseWorkspaces, error) {
	var response models.MappingWorkspaceUserRoleResponseWorkspaces

	query := `SELECT DISTINCT workspace_id FROM mapping_workspace_user_roles`
	var workspaceUniqueId []string
	err := tx.RawQuery(query).All(&workspaceUniqueId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, worksapceId := range workspaceUniqueId {
		resp, err := GetWorkspaceUserRoleMappingByWorkspaceId(tx, worksapceId)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		response = append(response, *resp)
	}
	return &response, nil
}

func GetWorkspaceUserRoleMappingByWorkspaceName(tx *pop.Connection, workspaceName string) (*models.MappingWorkspaceUserRoleResponseWorkspace, error) {
	var m models.MappingWorkspaceUserRoles
	var response models.MappingWorkspaceUserRoleResponseWorkspace

	workspace, err := GetWorkspaceByName(tx, workspaceName)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	response.Workspce = *workspace

	err = tx.Where("workspace_id = ?", workspace.ID).All(&m)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, userInfo := range m {
		role, err := GetRoleById(tx, userInfo.RoleID.String())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		u := &models.UserInfo{
			User: userInfo.User,
			Role: *role,
		}
		response.UserInfos = append(response.UserInfos, *u)
	}
	return &response, nil
}

func GetWorkspaceUserRoleMappingByWorkspaceId(tx *pop.Connection, workspaceId string) (*models.MappingWorkspaceUserRoleResponseWorkspace, error) {
	var m models.MappingWorkspaceUserRoles
	var response models.MappingWorkspaceUserRoleResponseWorkspace

	workspace, err := GetWorkspaceById(tx, workspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	response.Workspce = *workspace

	err = tx.Where("workspace_id = ?", workspace.ID).All(&m)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, userInfo := range m {
		role, err := GetRoleById(tx, userInfo.RoleID.String())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		u := &models.UserInfo{
			User: userInfo.User,
			Role: *role,
		}
		response.UserInfos = append(response.UserInfos, *u)
	}
	return &response, nil
}

func GetWorkspaceUserRoleMappingByUser(tx *pop.Connection, userId string) (*models.MappingWorkspaceUserRoleResponseUserArr, error) {
	var m models.MappingWorkspaceUserRoles
	var response models.MappingWorkspaceUserRoleResponseUserArr

	err := tx.Where("user_id = ?", userId).All(&m)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	fmt.Println(m)

	for _, userInfo := range m {
		var resp models.MappingWorkspaceUserRoleResponseUser
		role, err := GetRoleById(tx, userInfo.RoleID.String())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		resp.Role = *role

		worksapceProjectMapping, err := GetWorkspaceProjectMappingByWorkspaceId(tx, userInfo.WorkspaceID.String())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		resp.MappingWorkspaceProjectResponse = *worksapceProjectMapping
		response = append(response, resp)
	}

	return &response, nil
}

func DeleteWorkspaceUserRoleMapping(tx *pop.Connection, workspaceId string, userId string) error {
	var m models.MappingWorkspaceUserRole

	err := tx.Where("workspace_id = ? and user_id = ? ", workspaceId, userId).First(&m)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&m)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func DeleteWorkspaceUserRoleMappingByName(tx *pop.Connection, workspaceNmae string, userId string) error {
	workspaceId, err := GetWorkspaceIdByname(tx, workspaceNmae)
	if err != nil {
		log.Error(err)
		return err
	}
	err = DeleteWorkspaceUserRoleMapping(tx, workspaceId.String(), userId)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
