package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
)

func CreateRole(tx *pop.Connection, bindModel *models.MCIamRole) map[string]interface{} {

	err := tx.Create(bindModel)

	if err != nil {

	}
	return map[string]interface{}{
		"": "",
	}
}
