package handler

import (
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

func CreateWorkspace(tx *pop.Connection, bindModel *models.MCIamWorkspace) map[string]interface{} {

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

func GetWorkspaceList(tx *pop.Connection) *models.MCIamWorkspaces {
	bindModel := &models.MCIamWorkspaces{}

	err := tx.Eager().All(bindModel)

	if err != nil {

	}
	return bindModel
}

func GetWorkspace(tx *pop.Connection, wsId string) *models.MCIamWorkspace {
	ws := &models.MCIamWorkspace{}

	err := tx.Find(ws, wsId)
	if err != nil {

	}
	return ws
}

func DeleteWorkspace(tx *pop.Connection, wsId string) map[string]interface{} {
	ws := &models.MCIamWorkspace{}
	wsUuid, _ := uuid.FromString(wsId)
	ws.ID = wsUuid

	err := tx.Destroy(ws)
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
