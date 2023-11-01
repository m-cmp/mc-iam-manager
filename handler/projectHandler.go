package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
)

func CreateProject(tx *pop.Connection, bindModel *models.MCIamProject) map[string]interface{} {

	err := tx.Create(bindModel)

	if err != nil {

	}
	return map[string]interface{}{
		"": "",
	}
}
