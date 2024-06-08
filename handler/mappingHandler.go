package handler

import (
	"errors"
	"mc_iam_manager/models"
	"slices"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/opentracing/opentracing-go/log"
)

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
