package handler

import (
	"fmt"
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateWorkspaceProjectMappingByWorkspaceAndProject(tx *pop.Connection, mappingWorkspaceProject *models.MappingWorkspaceProject) (*models.MappingWorkspaceProject, error) {
	err := tx.Create(mappingWorkspaceProject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return mappingWorkspaceProject, nil
}

func CreateWorkspaceProjectMappingByWorkspaceAndProjectsName(tx *pop.Connection, mappingWorkspaceProjectsRequest *models.MappingWorkspaceProjectsNameRequest) (*models.MappingWorkspaceProjectResponse, error) {

	workspaceId, err := GetWorkspaceIdByname(tx, mappingWorkspaceProjectsRequest.WorkspaceName)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var errarr []error
	for _, projectName := range mappingWorkspaceProjectsRequest.ProjectNames {
		projectId, err := GetProjectIdByname(tx, projectName)
		if err != nil {
			log.Error(err)
			errarr = append(errarr, err)
			break
		}
		mappingWorkspaceProject := &models.MappingWorkspaceProject{
			WorkspaceID: workspaceId,
			ProjectID:   projectId,
		}
		_, err = CreateWorkspaceProjectMappingByWorkspaceAndProject(tx, mappingWorkspaceProject)
		if err != nil {
			log.Error(err)
		}
	}

	mappingWorkspaceProjectResponse, err := GetWorkspaceProjectMappingByWorkspaceName(tx, mappingWorkspaceProjectsRequest.WorkspaceName)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if len(errarr) > 0 {
		errstr := JoinErrors(errarr, "/")
		return mappingWorkspaceProjectResponse, fmt.Errorf(errstr)
	}

	return mappingWorkspaceProjectResponse, nil
}

func CreateWorkspaceProjectMappingByWorkspaceAndProjectsId(tx *pop.Connection, mappingWorkspaceProjectsRequest *models.MappingWorkspaceProjectsIdRequest) (*models.MappingWorkspaceProjectResponse, error) {
	var errarr []error
	for _, projectId := range mappingWorkspaceProjectsRequest.ProjectIds {
		workspaceUUId, _ := uuid.FromString(mappingWorkspaceProjectsRequest.WorkspaceId)
		projectUUId, _ := uuid.FromString(projectId)

		mappingWorkspaceProject := &models.MappingWorkspaceProject{
			WorkspaceID: workspaceUUId,
			ProjectID:   projectUUId,
		}
		_, err := CreateWorkspaceProjectMappingByWorkspaceAndProject(tx, mappingWorkspaceProject)
		if err != nil {
			log.Error(err)
		}
	}

	mappingWorkspaceProjectResponse, err := GetWorkspaceProjectMappingByWorkspaceId(tx, mappingWorkspaceProjectsRequest.WorkspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if len(errarr) > 0 {
		errstr := JoinErrors(errarr, "/")
		return mappingWorkspaceProjectResponse, fmt.Errorf(errstr)
	}

	return mappingWorkspaceProjectResponse, nil
}

func GetWorkspaceProjectMappingListByWorkspace(tx *pop.Connection) (*models.MappingWorkspaceProjectResponses, error) {
	query := `SELECT DISTINCT workspace_id FROM mapping_workspace_projects`
	var workspaceUniqueId []string
	err := tx.RawQuery(query).All(&workspaceUniqueId)
	if err != nil {
		log.Error(err)
	}

	var mappingWorkspaceProjectResponses models.MappingWorkspaceProjectResponses
	for _, workspaceId := range workspaceUniqueId {
		resp, err := GetWorkspaceProjectMappingByWorkspaceId(tx, workspaceId)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		mappingWorkspaceProjectResponses = append(mappingWorkspaceProjectResponses, *resp)
	}

	return &mappingWorkspaceProjectResponses, nil
}

func GetWorkspaceProjectMappingByWorkspaceName(tx *pop.Connection, workspaceName string) (*models.MappingWorkspaceProjectResponse, error) {
	var mppingWorkspaceProjectResponse models.MappingWorkspaceProjectResponse
	workspace, err := GetWorkspaceByName(tx, workspaceName)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	mppingWorkspaceProjectResponse.Workspace = *workspace

	var mappingWorkspaceProjects models.MappingWorkspaceProjects
	err = tx.Where("workspace_id = ?", workspace.ID).All(&mappingWorkspaceProjects)
	if err != nil {
		log.Error(err)
	}

	for _, mappingWorkspaceProject := range mappingWorkspaceProjects {
		project, err := GetProjectById(tx, mappingWorkspaceProject.ProjectID.String())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		mppingWorkspaceProjectResponse.Projects = append(mppingWorkspaceProjectResponse.Projects, *project)
	}

	return &mppingWorkspaceProjectResponse, nil
}

func GetWorkspaceProjectMappingByWorkspaceId(tx *pop.Connection, workspaceId string) (*models.MappingWorkspaceProjectResponse, error) {
	var mppingWorkspaceProjectResponse models.MappingWorkspaceProjectResponse
	workspace, err := GetWorkspaceById(tx, workspaceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	mppingWorkspaceProjectResponse.Workspace = *workspace

	var mappingWorkspaceProjects models.MappingWorkspaceProjects
	err = tx.Where("workspace_id = ?", workspace.ID).All(&mappingWorkspaceProjects)
	if err != nil {
		log.Error(err)
	}

	for _, mappingWorkspaceProject := range mappingWorkspaceProjects {
		project, err := GetProjectById(tx, mappingWorkspaceProject.ProjectID.String())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		mppingWorkspaceProjectResponse.Projects = append(mppingWorkspaceProjectResponse.Projects, *project)
	}

	return &mppingWorkspaceProjectResponse, nil
}

func DeleteWorkspaceProjectMappingByName(tx *pop.Connection, mappingWorkspaceProject *models.MappingWorkspaceProjectsDeleteNameRequest) error {
	workspaceId, err := GetWorkspaceIdByname(tx, mappingWorkspaceProject.WorkspaceName)
	if err != nil {
		log.Error(err)
		return err
	}
	projectId, err := GetProjectIdByname(tx, mappingWorkspaceProject.ProjectName)
	if err != nil {
		log.Error(err)
		return err
	}

	err = DeleteWorkspaceProjectMappingById(tx, workspaceId.String(), projectId.String())
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func DeleteWorkspaceProjectMappingById(tx *pop.Connection, workspaceID string, projectID string) error {
	target := models.MappingWorkspaceProject{}
	err := tx.Where("workspace_id = ? AND project_id = ?", workspaceID, projectID).First(&target)
	if err != nil {
		log.Error(err)
		return err
	}
	txerr := tx.Destroy(&target)
	if txerr != nil {
		log.Error(txerr)
		return txerr
	}
	return nil
}
