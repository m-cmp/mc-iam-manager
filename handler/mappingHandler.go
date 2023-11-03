package handler

import (
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func MappingWsUserRole(tx *pop.Connection, bindModel *models.MCIamWsUserRoleMapping) map[string]interface{} {
	if bindModel != nil {
		wsUserProjectModel := &models.MCIamWsUserRoleMapping{}

		wsId := bindModel.WsID
		roleId := bindModel.RoleID
		userId := bindModel.UserID

		q := tx.Eager().Where("ws_id = ?", wsId)
		q = q.Where("role_id = ?", roleId)
		q = q.Where("user_id = ?", userId)

		b, err := q.Exists(wsUserProjectModel)
		if err != nil {
			return map[string]interface{}{
				"error":  "something query error",
				"status": "301",
			}
		}

		if b {
			return map[string]interface{}{
				"error":  "already Exists",
				"status": "301",
			}
		}
	}
	LogPrintHandler("mapping ws user role bind model", bindModel)
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

func GetWsUserRole(tx *pop.Connection, bindModel *models.MCIamWsUserRoleMapping) *models.MCIamWsUserRoleMappings {

	respModel := &models.MCIamWsUserRoleMappings{}

	if user_id := bindModel.UserID; user_id != uuid.Nil {
		q := tx.Eager().Where("user_id = ?", user_id)
		err := q.All(respModel)
		if err != nil {

		}
	}

	if role_id := bindModel.RoleID; role_id != uuid.Nil {
		q := tx.Eager().Where("role_id = ?", role_id)
		err := q.All(respModel)
		if err != nil {

		}
	}
	if ws_id := bindModel.WsID; ws_id != uuid.Nil {
		q := tx.Eager().Where("ws_id = ?", ws_id)
		err := q.All(respModel)
		if err != nil {

		}
	}
	return respModel
}

func MappingWsProject(tx *pop.Connection, bindModel *models.MCIamWsProjectMapping) map[string]interface{} {
	// check dupe
	if bindModel != nil {
		wsPjModel := &models.MCIamWsProjectMapping{}

		wsId := bindModel.WsID
		projectId := bindModel.ProjectID

		q := tx.Eager().Where("ws_id = ?", wsId)
		q = q.Where("project_id = ?", projectId)
		b, err := q.Exists(wsPjModel)
		if err != nil {
			return map[string]interface{}{
				"message": "something query error",
				"status":  "301",
			}
		}

		if b {
			return map[string]interface{}{
				"message": "already Exists",
				"status":  "301",
			}
		}
	}
	LogPrintHandler("mapping ws project bind model", bindModel)
	err := tx.Create(bindModel)

	if err != nil {
		LogPrintHandler("mapping ws project error", err)

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

func MappingGetProjectByWorkspace(tx *pop.Connection, wsId string) *models.MCIamWsProjectMappings {
	ws := &models.MCIamWsProjectMappings{}

	err := tx.Eager().Where("ws_id =?", wsId).All(ws)
	if err != nil {

	}
	return ws

}

func MappingDeleteWsProject(tx *pop.Connection, bindModel *models.MCIamWsProjectMapping) map[string]interface{} {
	err := tx.Destroy(bindModel)
	if err != nil {
		return map[string]interface{}{
			"message": errors.WithStack(err),
			"status":  "301",
		}
	}
	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}

}

func MappingUserRole(tx *pop.Connection, bindModel *models.MCIamUserRoleMapping) map[string]interface{} {
	if bindModel != nil {
		userRoleModel := &models.MCIamUserRoleMapping{}

		roleId := bindModel.RoleID
		userId := bindModel.UserID

		q := tx.Eager().Where("role_id = ?", roleId)
		q = q.Where("user_id = ?", userId)

		b, err := q.Exists(userRoleModel)
		if err != nil {
			return map[string]interface{}{
				"error":  "something query error",
				"status": "301",
			}
		}

		if b {
			return map[string]interface{}{
				"error":  "already Exists",
				"status": "301",
			}
		}
	}
	LogPrintHandler("mapping user role bind model", bindModel)
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
