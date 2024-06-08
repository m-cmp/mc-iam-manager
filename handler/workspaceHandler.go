package handler

import (
	"errors"
	"mc_iam_manager/models"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/opentracing/opentracing-go/log"
)

func CreateWorkspace(tx *pop.Connection, s *models.MCIamWorkspace) (*models.MCIamWorkspace, error) {
	workspacExist, err := IsExistsWorkspace(tx, s.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if workspacExist {
		return nil, errors.New("workspace is duplicated")
	}
	createerr := tx.Create(s)
	if createerr != nil {
		log.Error(createerr)
		return nil, createerr
	}
	return s, nil
}

func IsExistsWorkspace(tx *pop.Connection, workspaceId string) (bool, error) {
	var workspace models.MCIamWorkspace
	txerr := tx.Where("workspace_id = ?", workspaceId).First(&workspace)
	if txerr != nil {
		if strings.Contains(txerr.Error(), "no rows in result set") {
			return false, nil
		}
		return false, txerr
	}
	return true, nil
}

func GetWorkspaceList(tx *pop.Connection) (*models.MCIamWorkspaces, error) {
	var workspaces models.MCIamWorkspaces
	err := tx.All(&workspaces)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &workspaces, nil
}

func GetWorkspace(tx *pop.Connection, workspaceId string) (*models.MCIamWorkspace, error) {
	var workspace models.MCIamWorkspace
	err := tx.Where("workspace_id = ?", workspaceId).First(&workspace)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &workspace, nil
}

func UpdateWorkspace(tx *pop.Connection, workspace *models.MCIamWorkspace) (*models.MCIamWorkspace, error) {
	var targetworkspace models.MCIamWorkspace
	err := tx.Where("workspace_id = ?", workspace.WorkspaceID).First(&targetworkspace)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetworkspace.Description = workspace.Description
	err = tx.Update(&targetworkspace)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &targetworkspace, nil
}

func DeleteWorkspace(tx *pop.Connection, workspaceId string) error {
	var workspace models.MCIamWorkspace
	err := tx.Where("workspace_id = ?", workspaceId).First(&workspace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&workspace)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
