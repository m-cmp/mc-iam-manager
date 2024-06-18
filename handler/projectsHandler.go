package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateProject(tx *pop.Connection, s *models.Project) (*models.Project, error) {
	err := tx.Create(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func GetProjectList(tx *pop.Connection) (*models.Projects, error) {
	var s models.Projects
	err := tx.All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func GetProjectById(tx *pop.Connection, id uuid.UUID) (*models.Project, error) {
	var s models.Project
	err := tx.Find(&s, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func SearchProjectsByName(tx *pop.Connection, name string, option string) (*models.Projects, error) {
	table := "projects"
	var query string
	if option == "contain" {
		query = "SELECT * FROM " + table + " WHERE name ILIKE  '%" + name + "%';"
	} else {
		query = "SELECT * FROM " + table + " WHERE name LIKE  '" + name + "';"
	}
	var s models.Projects
	err := tx.RawQuery(query).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func UpdateProject(tx *pop.Connection, s *models.Project) (*models.Project, error) {
	err := tx.Update(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func DeleteProject(tx *pop.Connection, s *models.Project) error {
	err := tx.Destroy(s)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
