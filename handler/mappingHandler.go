package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
)

func WsUserRoleMapping(tx *pop.Connection, bindModel *models.MCIamWsUserRoleMapping) map[string]interface{} {

	err := tx.Create(bindModel)

	if err != nil {

	}
	return map[string]interface{}{
		"": "",
	}
}
