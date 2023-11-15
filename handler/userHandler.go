package handler

import (
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

func CreateUser(tx *pop.Connection, bindModel *models.MCIamProject) map[string]interface{} {

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

func GetUserList(tx *pop.Connection) []models.UserEntity {
	bindModel := []models.UserEntity{}

	err := tx.Eager().All(&bindModel)

	if err != nil {

	}
	return bindModel
}

func GetUser(tx *pop.Connection, projectId string) *models.UserEntity {
	ws := &models.UserEntity{}

	err := tx.Find(ws, projectId)
	if err != nil {

	}
	return ws
}

func DeleteUser(tx *pop.Connection, wsId string) map[string]interface{} {
	ws := &models.MCIamProject{}
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
