package handler

import (
	"mc_iam_manager/models"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateWPmapping(tx *pop.Connection, s *models.WorkspaceProjectMapping) (*models.WorkspaceProjectMapping, error) {
	err := tx.Create(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func GetWPmappingList(tx *pop.Connection) (*models.WorkspaceProjectMappings, error) {
	var s models.WorkspaceProjectMappings
	err := tx.All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

type getWPmappingListOrderbyWorkspaceResp struct {
	Workspace models.Workspace `json:"workspace"`
	Projects  []models.Project `json:"projects"`
}
type getWPmappingListOrderbyWorkspaceResps []getWPmappingListOrderbyWorkspaceResp

type getWPmappingListOrderbyWorkspace struct {
	WorkspaceId string `db:"workspace_id"`
	ProjectIds  string `db:"projects"`
}
type getWPmappingListOrderbyWorkspaces []getWPmappingListOrderbyWorkspace

func GetWPmappingListOrderbyWorkspace(tx *pop.Connection) (*getWPmappingListOrderbyWorkspaceResps, error) {
	var resp getWPmappingListOrderbyWorkspaceResps
	query := "SELECT workspace_id, array_agg(project_id) AS projects FROM workspace_project_mappings GROUP BY workspace_id;"
	var s getWPmappingListOrderbyWorkspaces
	err := tx.RawQuery(query).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err

	}
	for _, ss := range s {
		var res getWPmappingListOrderbyWorkspaceResp
		ws, err := GetWorkspaceById(tx, uuid.FromStringOrNil(ss.WorkspaceId))
		if err != nil {
			log.Error(err)
			return nil, err
		}
		res.Workspace = *ws
		for _, projectid := range extractAndSplit(ss.ProjectIds) {
			prj, err := GetProjectById(tx, uuid.FromStringOrNil(projectid))
			if err != nil {
				log.Error(err)
				return nil, err
			}
			res.Projects = append(res.Projects, *prj)
		}
		resp = append(resp, res)
	}
	return &resp, err
}

func GetWPmappingListByWorkspaceId(tx *pop.Connection, worksapceId uuid.UUID) (*getWPmappingListOrderbyWorkspaceResp, error) {
	var err error
	var s models.WorkspaceProjectMappings
	var resp getWPmappingListOrderbyWorkspaceResp

	ws, err := GetWorkspaceById(tx, worksapceId)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	resp.Workspace = *ws

	err = tx.Where("workspace_id = ?", worksapceId).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, ss := range s {
		prj, err := GetProjectById(tx, ss.ProjectID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		resp.Projects = append(resp.Projects, *prj)
	}
	return &resp, nil
}

func GetWPmappingById(tx *pop.Connection, workspaceId string, projectId string) (*models.WorkspaceProjectMapping, error) {
	m := &models.WorkspaceProjectMapping{}
	err := tx.Where("workspace_id = ? and project_id = ?", workspaceId, projectId).First(m)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return m, nil
}

func GetWPmappingByProjectId(tx *pop.Connection, projectId string) (*models.WorkspaceProjectMapping, error) {
	var s models.WorkspaceProjectMapping
	err := tx.Where("project_id = ? ", projectId).First(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

type updateWPmappings struct {
	WorkspaceId string `db:"workspace_id"`
	ProjectIds  string `db:"projects"`
}

func UpdateWPmappings(tx *pop.Connection, worksapceId string, projectIds []string) (*getWPmappingListOrderbyWorkspaceResp, error) {
	query := "SELECT workspace_id, array_agg(project_id) AS projects FROM workspace_project_mappings  WHERE workspace_id = '" + worksapceId + "' GROUP BY workspace_id;"
	var s updateWPmappings
	err := tx.RawQuery(query).First(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	_, projectTodel, projectToCreate := compareStringArrays(extractAndSplit(s.ProjectIds), projectIds)
	for _, projectId := range projectTodel {
		workspace, err := GetWPmappingById(tx, worksapceId, projectId)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		err = DeleteWPmapping(tx, workspace)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}
	for _, projectId := range projectToCreate {
		m := &models.WorkspaceProjectMapping{
			WorkspaceID: uuid.FromStringOrNil(worksapceId),
			ProjectID:   uuid.FromStringOrNil(projectId),
		}
		_, err := CreateWPmapping(tx, m)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}
	resp, err := GetWPmappingListByWorkspaceId(tx, uuid.FromStringOrNil(worksapceId))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return resp, nil
}

func DeleteWPmapping(tx *pop.Connection, s *models.WorkspaceProjectMapping) error {
	err := tx.Destroy(s)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func extractAndSplit(input string) []string {
	input = strings.Trim(input, "{}")
	result := strings.Split(input, ",")
	return result
}

func compareStringArrays(arr1, arr2 []string) (common, onlyInArr1, onlyInArr2 []string) {
	// Create maps to store the presence of elements
	map1 := make(map[string]bool)
	map2 := make(map[string]bool)

	// Populate the maps
	for _, item := range arr1 {
		map1[item] = true
	}
	for _, item := range arr2 {
		map2[item] = true
	}

	// Find common elements and elements only in arr1
	for _, item := range arr1 {
		if map2[item] {
			common = append(common, item)
		} else {
			onlyInArr1 = append(onlyInArr1, item)
		}
	}

	// Find elements only in arr2
	for _, item := range arr2 {
		if !map1[item] {
			onlyInArr2 = append(onlyInArr2, item)
		}
	}

	return common, onlyInArr1, onlyInArr2
}
