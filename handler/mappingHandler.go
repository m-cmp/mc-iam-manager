package handler

import (
	"errors"
	"mc_iam_manager/models"
	"slices"

	"strconv"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/opentracing/opentracing-go/log"
)

/////////////////////////////
// WorkspaceProjectMapping //
/////////////////////////////

func CreateWorkspaceProjectMapping(tx *pop.Connection, mappingWorkspaceProjectRequest *models.MCIamMappingWorkspaceProjectRequest) (*models.MCIamMappingWorkspaceProjectResponse, error) {
	for _, prjid := range mappingWorkspaceProjectRequest.Projects {
		projectExist, err := IsExistsProject(tx, prjid)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if !projectExist {
			return nil, errors.New("project is not exist")
		}
	}
	workspaceExist, err := IsExistsWorkspace(tx, mappingWorkspaceProjectRequest.WorkspaceID)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if !workspaceExist {
		return nil, errors.New("workspace is not exist")
	}

	for _, prjid := range mappingWorkspaceProjectRequest.Projects {
		mappingWorkspaceProject := &models.MCIamMappingWorkspaceProject{
			WorkspaceID: mappingWorkspaceProjectRequest.WorkspaceID,
			ProjectID:   prjid,
		}

		workspaceProjectMappingExist, err := IsExistsWorkspaceProjectMapping(tx, mappingWorkspaceProjectRequest.WorkspaceID, mappingWorkspaceProject.ProjectID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if workspaceProjectMappingExist {
			err := errors.New("workspaceProjectMappingExist is exist")
			log.Error(err)
		} else {
			createerr := tx.Create(mappingWorkspaceProject)
			if createerr != nil {
				log.Error(createerr)
				return nil, createerr
			}
		}
	}

	resp, err := GetWorkspaceProjectMapping(tx, mappingWorkspaceProjectRequest.WorkspaceID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return resp, nil
}

func IsExistsWorkspaceProjectMapping(tx *pop.Connection, workspaceId string, projectId string) (bool, error) {
	var mappingWorkspaceProject models.MCIamMappingWorkspaceProject
	txerr := tx.Where("workspace_id = ? AND project_id = ?", workspaceId, projectId).First(&mappingWorkspaceProject)
	if txerr != nil {
		if strings.Contains(txerr.Error(), "no rows in result set") {
			return false, nil
		}
		return false, txerr
	}
	return true, nil
}

func GetWorkspaceProjectMappingList(tx *pop.Connection) (*[]models.MCIamMappingWorkspaceProjectResponse, error) {
	response := []models.MCIamMappingWorkspaceProjectResponse{}

	worspaceList, err := GetWorkspaceList(tx)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	for _, worspace := range *worspaceList {
		mapping, err := GetWorkspaceProjectMapping(tx, worspace.WorkspaceID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if len(mapping.Projects) > 0 {
			response = append(response, *mapping)
		}
	}

	return &response, nil
}

func GetWorkspaceProjectMapping(tx *pop.Connection, workspaceId string) (*models.MCIamMappingWorkspaceProjectResponse, error) {
	response := &models.MCIamMappingWorkspaceProjectResponse{}

	workspace, err := GetWorkspace(tx, workspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var mappingWorkspaceProjects models.MCIamMappingWorkspaceProjects
	txerr := tx.Where("workspace_id = ?", workspaceId).All(&mappingWorkspaceProjects)
	if txerr != nil {
		log.Error(txerr)
		return nil, txerr
	}

	response.Workspace = *workspace
	for _, mappingWorkspaceProject := range mappingWorkspaceProjects {
		prj, err := GetProject(tx, string(mappingWorkspaceProject.ProjectID))
		if err != nil {
			log.Error(err)
			return nil, err
		}
		response.Projects = append(response.Projects, *prj)
	}

	return response, nil
}

func UpdateWorkspaceProjectMapping(tx *pop.Connection, mappingWorkspaceProjectRequest *models.MCIamMappingWorkspaceProjectRequest) (*models.MCIamMappingWorkspaceProjectResponse, error) {
	workspaceProjectMapping, err := GetWorkspaceProjectMapping(tx, mappingWorkspaceProjectRequest.WorkspaceID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, prj := range workspaceProjectMapping.Projects {
		if !slices.Contains(mappingWorkspaceProjectRequest.Projects, prj.ProjectID) {
			err := DeleteWorkspaceProjectMapping(tx, mappingWorkspaceProjectRequest.WorkspaceID, prj.ProjectID)
			if err != nil {
				log.Error(err)
				return nil, err
			}
		}
	}

	workspaceProjectMappinginput := &models.MCIamMappingWorkspaceProjectRequest{
		WorkspaceID: mappingWorkspaceProjectRequest.WorkspaceID,
		Projects:    mappingWorkspaceProjectRequest.Projects,
	}

	resp, err := CreateWorkspaceProjectMapping(tx, workspaceProjectMappinginput)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return resp, nil
}

func DeleteWorkspaceProjectMapping(tx *pop.Connection, workspaceId string, projectId string) error {
	var mappingWorkspaceProject models.MCIamMappingWorkspaceProject
	err := tx.Where("workspace_id = ? AND project_id = ?", workspaceId, projectId).First(&mappingWorkspaceProject)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&mappingWorkspaceProject)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func DeleteWorkspaceProjectMappingAllByWorkspace(tx *pop.Connection, workspaceId string) error {
	var mappingWorkspaceProjects models.MCIamMappingWorkspaceProjects
	err := tx.Where("workspace_id = ?", workspaceId).All(&mappingWorkspaceProjects)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&mappingWorkspaceProjects)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func DeleteWorkspaceProjectMappingByProject(tx *pop.Connection, projectId string) error {
	var mappingWorkspaceProjects models.MCIamMappingWorkspaceProjects
	err := tx.Where("project_id = ?", projectId).All(&mappingWorkspaceProjects)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&mappingWorkspaceProjects)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// ///////////////////////////
// WorkspaceUserRoleMapping //
// ///////////////////////////

func CreateWorkspaceUserRoleMapping(tx *pop.Connection, mappingWorkspaceUserRole *models.MCIamMappingWorkspaceUserRole) (*models.MCIamMappingWorkspaceUserRole, error) {
	isworkspace, err := IsExistsWorkspace(tx, mappingWorkspaceUserRole.WorkspaceID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	isrole, err := IsExistsRole(tx, mappingWorkspaceUserRole.RoleName)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !isworkspace || !isrole {
		errmsg := "workspace is " + strconv.FormatBool(isworkspace) + " role is " + strconv.FormatBool(isrole) + " check input."
		err := errors.New(errmsg)
		return nil, err
	}

	isWorkspaceUserRole, err := IsExistsWorkspaceUserRoleMapping(tx, mappingWorkspaceUserRole.WorkspaceID, mappingWorkspaceUserRole.RoleName, mappingWorkspaceUserRole.UserID)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if isWorkspaceUserRole {
		return nil, errors.New("WorkspaceUserRoleMapping is duplicated")
	}

	createerr := tx.Create(mappingWorkspaceUserRole)
	if createerr != nil {
		log.Error(createerr)
		return nil, createerr
	}

	return mappingWorkspaceUserRole, nil
}

func IsExistsWorkspaceUserRoleMapping(tx *pop.Connection, workspaceId string, roleId string, userId string) (bool, error) {
	var mappingWorkspaceUserRole models.MCIamMappingWorkspaceUserRole
	txerr := tx.Where("workspace_id = ? AND role_name = ? AND user_id = ?", workspaceId, roleId, userId).First(&mappingWorkspaceUserRole)
	if txerr != nil {
		if strings.Contains(txerr.Error(), "no rows in result set") {
			return false, nil
		}
		return false, txerr
	}
	return true, nil
}

func GetWorkspaceUserRoleMapping(tx *pop.Connection) (*[]models.MCIamMappingWorkspaceUserRoleListResponse, error) {
	response := []models.MCIamMappingWorkspaceUserRoleListResponse{}

	workspaces, err := GetWorkspaceList(tx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, workspace := range *workspaces {
		userRoleMapping, err := GetWorkspaceUserRoleMappingByWorkspace(tx, workspace.Name)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		response = append(response, *userRoleMapping)
	}

	return &response, nil
}

func GetWorkspaceUserRoleMappingByWorkspace(tx *pop.Connection, workspaceId string) (*models.MCIamMappingWorkspaceUserRoleListResponse, error) {
	response := models.MCIamMappingWorkspaceUserRoleListResponse{}

	workspace, err := GetWorkspace(tx, workspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	response.Workspace = *workspace

	var mappingWorkspaceUserRoles models.MCIamMappingWorkspaceUserRoles
	txerr := tx.Where("workspace_id = ?", workspaceId).All(&mappingWorkspaceUserRoles)
	if txerr != nil {
		log.Error(txerr)
		return nil, txerr
	}

	for _, mappingWorkspaceUserRole := range mappingWorkspaceUserRoles {
		user := models.UserRoleMappingResponse{}
		user.UserID = mappingWorkspaceUserRole.UserID
		user.RoleName = mappingWorkspaceUserRole.RoleName
		response.Users = append(response.Users, user)
	}

	return &response, nil
}

func GetWorkspaceUserRoleMappingByWorkspaceUser(tx *pop.Connection, workspaceId string, userId string) (*models.MCIamMappingWorkspaceUserRoleUserResponse, error) {
	response := models.MCIamMappingWorkspaceUserRoleListResponse{}

	workspace, err := GetWorkspace(tx, workspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	response.Workspace = *workspace

	var mappingWorkspaceUserRole models.MCIamMappingWorkspaceUserRole
	txerr := tx.Where("workspace_id = ? AND user_id = ?", workspaceId, userId).First(&mappingWorkspaceUserRole)
	if txerr != nil {
		log.Error(txerr)
		return nil, txerr
	}

	m := models.MCIamMappingWorkspaceUserRoleUserResponse{}
	m.RoleName = mappingWorkspaceUserRole.RoleName
	m.Workspace = *workspace
	mappingWorkspaceProject, err := GetWorkspaceProjectMapping(tx, mappingWorkspaceUserRole.WorkspaceID)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	m.Project = mappingWorkspaceProject.Projects

	return &m, nil
}

func GetWorkspaceUserRoleMappingByUser(tx *pop.Connection, userId string) (*models.MCIamMappingWorkspaceUserRoleUserResponses, error) {
	response := models.MCIamMappingWorkspaceUserRoleUserResponses{}

	var mappingWorkspaceUserRoles models.MCIamMappingWorkspaceUserRoles
	txerr := tx.Where("user_id = ?", userId).All(&mappingWorkspaceUserRoles)
	if txerr != nil {
		log.Error(txerr)
		return nil, txerr
	}

	for _, mappingWorkspaceUserRole := range mappingWorkspaceUserRoles {
		mapping := models.MCIamMappingWorkspaceUserRoleUserResponse{}
		mapping.RoleName = mappingWorkspaceUserRole.RoleName
		mappingWorkspaceProject, err := GetWorkspaceProjectMapping(tx, mappingWorkspaceUserRole.WorkspaceID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		mapping.Workspace = mappingWorkspaceProject.Workspace
		mapping.Project = mappingWorkspaceProject.Projects
		response = append(response, mapping)
	}

	return &response, nil
}

func UpdateWorkspaceUserRoleMapping(tx *pop.Connection, workspaceId string, userId string, mappingWorkspaceUserRole *models.MCIamMappingWorkspaceUserRole) (*models.MCIamMappingWorkspaceUserRoleUserResponse, error) {

	isworkspace, err := IsExistsWorkspace(tx, workspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	isrole, err := IsExistsRole(tx, mappingWorkspaceUserRole.RoleName)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !isworkspace || !isrole {
		errmsg := "workspace is " + strconv.FormatBool(isworkspace) + " role is " + strconv.FormatBool(isrole) + " check input."
		err := errors.New(errmsg)
		return nil, err
	}

	var targetProject models.MCIamMappingWorkspaceUserRole
	txerr := tx.Where("workspace_id = ? AND user_id = ?", workspaceId, userId).First(&targetProject)
	if txerr != nil {
		log.Error(txerr)
		return nil, txerr
	}

	targetProject.RoleName = mappingWorkspaceUserRole.RoleName
	txerr = tx.Update(&targetProject)
	if txerr != nil {
		log.Error(txerr)
		return nil, txerr
	}

	resp, err := GetWorkspaceUserRoleMappingByWorkspaceUser(tx, workspaceId, userId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return resp, nil
}

func DeleteWorkspaceUserRoleMapping(tx *pop.Connection, workspaceId string, userId string) error {
	var mappingWorkspaceUserRole models.MCIamMappingWorkspaceUserRole
	err := tx.Where("workspace_id = ? AND user_id = ?", workspaceId, userId).First(&mappingWorkspaceUserRole)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&mappingWorkspaceUserRole)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func DeleteWorkspaceUserRoleMappingAll(tx *pop.Connection, workspaceId string) error {
	var mappingWorkspaceUserRoles models.MCIamMappingWorkspaceUserRoles
	err := tx.Where("workspace_id = ?", workspaceId).All(&mappingWorkspaceUserRoles)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&mappingWorkspaceUserRoles)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
