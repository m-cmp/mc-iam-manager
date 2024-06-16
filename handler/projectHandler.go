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
	var Projects models.Projects
	err := tx.All(&Projects)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Projects, nil
}

func GetProjectByName(tx *pop.Connection, name string) (*models.Project, error) {
	var project models.Project
	err := tx.Where("name = ?", name).First(&project)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &project, nil
}

func GetProjectById(tx *pop.Connection, Id string) (*models.Project, error) {
	var Project models.Project
	err := tx.Find(&Project, Id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Project, nil
}

func GetProjectIdByname(tx *pop.Connection, name string) (uuid.UUID, error) {
	var targetProject models.Project
	err := tx.Where("name = ?", name).First(&targetProject)
	if err != nil {
		log.Error(err)
		return uuid.Nil, err
	}
	return targetProject.ID, nil
}

func UpdateProjectByname(tx *pop.Connection, name string, src *models.Project) (*models.Project, error) {
	targetProject, err := GetProjectByName(tx, name)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetProject.Name = src.Name
	targetProject.Description = src.Description

	err = tx.Update(targetProject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return targetProject, nil
}

func UpdateProjectById(tx *pop.Connection, id string, src *models.Project) (*models.Project, error) {
	targetProject, err := GetProjectById(tx, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetProject.Name = src.Name
	targetProject.Description = src.Description

	err = tx.Update(targetProject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return targetProject, nil
}

func DeleteProjectByName(tx *pop.Connection, name string) error {
	Project, err := GetProjectByName(tx, name)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(Project)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func DeleteProjectById(tx *pop.Connection, id string) error {
	Project, err := GetProjectById(tx, id)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(Project)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
