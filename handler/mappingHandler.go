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

func MappingWsUser(tx *pop.Connection, bindModel *models.MCIamWsUserMapping) map[string]interface{} {
	if bindModel != nil {
		wsUserModel := &models.MCIamWsUserMapping{}

		wsId := bindModel.WsID
		userId := bindModel.UserID

		q := tx.Eager().Where("ws_id = ?", wsId)
		q = q.Where("user_id = ?", userId)

		b, err := q.Exists(wsUserModel)
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
	LogPrintHandler("mapping ws user bind model", bindModel)
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

func MappingGetProjectByWorkspace(tx *pop.Connection, wsId string) *models.ParserWsProjectMapping {
	ws := []models.MCIamWsProjectMapping{}
	parsingWs := &models.ParserWsProjectMapping{}

	q := tx.Eager().Where("ws_id =?", wsId)
	b, err := q.Exists(ws)
	if err != nil {

	}
	if b {
		err := q.All(&ws)
		if err != nil {

		}

		parsingWs = ParserWsProjectByWs(ws, wsId)
	}

	return parsingWs

}

func MappingWsProjectValidCheck(tx *pop.Connection, wsId string, projectId string) map[string]interface{} {
	ws := &models.MCIamWsProjectMapping{}

	q := tx.Eager().Where("ws_id =?", wsId)
	q = q.Where("project_id =?", projectId)

	b, _ := q.Exists(ws)
	// if err != nil {
	// 	return map[string]interface{}{
	// 		"error":  "something query error",
	// 		"status": "301",
	// 	}
	// }

	if b {
		project := GetProject(tx, projectId)
		return map[string]interface{}{
			"message": "valid",
			"project": project,
		}
	}
	return map[string]interface{}{
		"message": "invalid",
		"error":   "invalid project",
		"status":  "301",
	}

}

func ParserWsProjectByWs(bindModels []models.MCIamWsProjectMapping, ws_id string) *models.ParserWsProjectMapping {
	parserWsProject := &models.ParserWsProjectMapping{}
	projectArray := []models.MCIamProject{}
	wsUuid, _ := uuid.FromString(ws_id)
	LogPrintHandler("#### bindmodels ####", bindModels)
	for _, obj := range bindModels {
		LogPrintHandler("#### wsuuid ####", obj.WsID)
		if wsUuid == obj.WsID {
			parserWsProject.WsID = obj.WsID
			parserWsProject.Ws = obj.Ws
			if obj.ProjectID != uuid.Nil {
				projectArray = append(projectArray, *obj.Project)
				parserWsProject.Projects = projectArray
			}
		}

	}
	return parserWsProject
}

// func ParserWsProject(tx *pop.Connection, bindModels []models.MCIamWorkspace) *models.ParserWsProjectMappings {
// 	parserWsProject := []models.ParserWsProjectMapping{}
// 	projectArray := []models.MCIamProject{}

// 	ParserWsProjectByWs

// 	return parserWsProject
// }

func MappingUserRole(tx *pop.Connection, bindModel *models.MCIamUserRoleMapping) map[string]interface{} {
	if bindModel.ID != uuid.Nil {
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
