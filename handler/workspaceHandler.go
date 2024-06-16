package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateWorkspace(tx *pop.Connection, s *models.Workspace) (*models.Workspace, error) {
	err := tx.Create(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func GetWorkspaceList(tx *pop.Connection) (*models.Workspaces, error) {
	var workspaces models.Workspaces
	err := tx.All(&workspaces)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &workspaces, nil
}

func GetWorkspaceByName(tx *pop.Connection, name string) (*models.Workspace, error) {
	var workspace models.Workspace
	err := tx.Where("name = ?", name).First(&workspace)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &workspace, nil
}

func GetWorkspaceById(tx *pop.Connection, Id string) (*models.Workspace, error) {
	var workspace models.Workspace
	err := tx.Find(&workspace, Id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &workspace, nil
}

func GetWorkspaceIdByname(tx *pop.Connection, name string) (uuid.UUID, error) {
	var targetworkspace models.Workspace
	err := tx.Where("name = ?", name).First(&targetworkspace)
	if err != nil {
		log.Error(err)
		return uuid.Nil, err
	}
	return targetworkspace.ID, nil
}

func UpdateWorkspaceByname(tx *pop.Connection, name string, src *models.Workspace) (*models.Workspace, error) {
	targetworkspace, err := GetWorkspaceByName(tx, name)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetworkspace.Name = src.Name
	targetworkspace.Description = src.Description

	err = tx.Update(targetworkspace)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return targetworkspace, nil
}

func UpdateWorkspaceById(tx *pop.Connection, id string, src *models.Workspace) (*models.Workspace, error) {
	targetworkspace, err := GetWorkspaceById(tx, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetworkspace.Name = src.Name
	targetworkspace.Description = src.Description

	err = tx.Update(targetworkspace)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return targetworkspace, nil
}

func DeleteWorkspaceByName(tx *pop.Connection, name string) error {
	workspace, err := GetWorkspaceByName(tx, name)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(workspace)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func DeleteWorkspaceById(tx *pop.Connection, id string) error {
	workspace, err := GetWorkspaceById(tx, id)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(workspace)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
