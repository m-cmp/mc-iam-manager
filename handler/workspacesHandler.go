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
	var s models.Workspaces
	err := tx.All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func GetWorkspaceById(tx *pop.Connection, id uuid.UUID) (*models.Workspace, error) {
	var s models.Workspace
	err := tx.Find(&s, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func SearchWorkspacesByName(tx *pop.Connection, name string, option string) (*models.Workspaces, error) {
	table := "workspaces"
	var query string
	if option == "contain" {
		query = "SELECT * FROM " + table + " WHERE name ILIKE  '%" + name + "%';"
	} else {
		query = "SELECT * FROM " + table + " WHERE name LIKE  '" + name + "';"
	}
	var s models.Workspaces
	err := tx.RawQuery(query).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func UpdateWorkspace(tx *pop.Connection, s *models.Workspace) (*models.Workspace, error) {
	err := tx.Update(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func DeleteWorkspace(tx *pop.Connection, s *models.Workspace) error {
	err := tx.Destroy(s)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
