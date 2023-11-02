package handler

import (
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

func CreateProject(tx *pop.Connection, bindModel *models.MCIamProject) map[string]interface{} {

	err := tx.Create(bindModel)

	if err != nil {
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}
}

func GetProjectList(tx *pop.Connection) *models.MCIamProjects {
	bindModel := &models.MCIamProjects{}

	err := tx.Eager().All(bindModel)

	if err != nil {

	}
	return bindModel
}

func GetProject(tx *pop.Connection, wsId string) *models.MCIamProject {
	ws := &models.MCIamProject{}

	err := tx.Find(ws, wsId)
	if err != nil {

	}
	return ws
}

func DeleteProject(tx *pop.Connection, wsId string) map[string]interface{} {
	ws := &models.MCIamProject{}
	wsUuid, _ := uuid.FromString(wsId)
	ws.ID = wsUuid

	err := tx.Destroy(ws)
	if err != nil {

	}
	return map[string]interface{}{
		"": "",
	}
}
