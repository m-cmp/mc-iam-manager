package handler

import (
	"errors"
	"mc_iam_manager/models"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/opentracing/opentracing-go/log"
)

func CreateProject(tx *pop.Connection, s *models.MCIamProject) (*models.MCIamProject, error) {
	workspacExist, err := IsExistsProjectDB(tx, s.ProjectID)
	if err != nil {
		return nil, err
	}
	if workspacExist {
		return nil, errors.New("project is duplicated")
	}
	createerr := tx.Create(s)
	if createerr != nil {
		log.Error(createerr)
		return nil, createerr
	}
	return s, nil
}

func IsExistsProjectDB(tx *pop.Connection, projectId string) (bool, error) {
	var project models.MCIamProject
	txerr := tx.Where("project_id = ?", projectId).First(&project)
	if txerr != nil {
		if strings.Contains(txerr.Error(), "no rows in result set") {
			return false, nil
		}
		return false, txerr
	}
	return true, nil
}

func GetProjectList(tx *pop.Connection) (*models.MCIamProjects, error) {
	var projects models.MCIamProjects
	err := tx.All(&projects)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &projects, nil
}

func GetProject(tx *pop.Connection, ProjectId string) (*models.MCIamProject, error) {
	var project models.MCIamProject
	err := tx.Where("project_id = ?", ProjectId).First(&project)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &project, nil
}

func UpdateProject(tx *pop.Connection, Project *models.MCIamProject) (*models.MCIamProject, error) {
	var targetProject models.MCIamProject
	err := tx.Where("project_id = ?", Project.ProjectID).First(&targetProject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetProject.Description = Project.Description
	err = tx.Update(&targetProject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &targetProject, nil
}

func DeleteProject(tx *pop.Connection, ProjectId string) error {
	var Project models.MCIamProject
	err := tx.Where("project_id = ?", ProjectId).First(&Project)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&Project)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
