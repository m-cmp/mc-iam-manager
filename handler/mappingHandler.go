package handler

import (
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func MappingWsUserRole(tx *pop.Connection, bindModel *models.MCIamMappingWorkspaceUserRoles) (models.MCIamMappingWorkspaceUserRoles, error) {
	if bindModel != nil {
		cblogger.Info("bindModel")
		cblogger.Info(bindModel)

		for _, mapping := range *bindModel {
			wsUserMapper := &models.MCIamMappingWorkspaceUserRole{}

			wsId := mapping.WorkspaceID
			roleId := mapping.RoleName
			userId := mapping.UserID

			q := tx.Eager().Where("workspace_id = ?", wsId)
			q = q.Where("role_name = ?", roleId)
			q = q.Where("user_id = ?", userId)

			b, err := q.Exists(wsUserMapper)
			if err != nil {
				cblogger.Errorf("Error bind, %s, %s, %s", wsId, userId, roleId)
				return nil, err
			}

			if b {
				cblogger.Error("Already mapped ws user, ?, ?, ?", wsId, userId, roleId)
				return nil, errors.Wrap(err, "Already mapped ws user")
			}
		}

		LogPrintHandler("mapping ws user role bind model", bindModel)

		err := tx.Create(bindModel)

		if err != nil {
			cblogger.Error("Create Err, ?", err)
			return nil, err
		}
	}

	return *bindModel, nil

}

//func MappingWsUser(tx *pop.Connection, bindModel *models.MCIamWsUserMapping) map[string]interface{} {
//	if bindModel != nil {
//		wsUserModel := &models.MCIamWsUserRoleMapping{}
//
//		wsId := bindModel.WsID
//		userId := bindModel.UserID
//
//		q := tx.Eager().Where("ws_id = ?", wsId)
//		q = q.Where("user_id = ?", userId)
//
//		b, err := q.Exists(wsUserModel)
//		if err != nil {
//			return map[string]interface{}{
//				"error":  "something query error",
//				"status": "301",
//			}
//		}
//
//		if b {
//			return map[string]interface{}{
//				"error":  "already Exists",
//				"status": "301",
//			}
//		}
//	}
//	LogPrintHandler("mapping ws user bind model", bindModel)
//	err := tx.Create(bindModel)
//
//	if err != nil {
//		return map[string]interface{}{
//			"message": err,
//			"status":  http.StatusBadRequest,
//		}
//	}
//	return map[string]interface{}{
//		"message": "success",
//		"status":  http.StatusOK,
//	}
//}

func GetWsUserRole(tx *pop.Connection, userId string) *models.MCIamMappingWorkspaceUserRole {

	respModel := &models.MCIamMappingWorkspaceUserRole{}

	if userId != "" {
		q := tx.Eager().Where("user_id = ?", userId)
		err := q.All(respModel)
		if err != nil {

		}
	}

	// if role_id := bindModel.RoleID; role_id != uuid.Nil {
	// 	q := tx.Eager().Where("role_id = ?", role_id)
	// 	err := q.All(respModel)
	// 	if err != nil {

	// 	}
	// }
	// if ws_id := bindModel.WsID; ws_id != uuid.Nil {
	// 	q := tx.Eager().Where("ws_id = ?", ws_id)
	// 	err := q.All(respModel)
	// 	if err != nil {

	// 	}
	// }
	return respModel
}

func AttachProjectToWorkspace(tx *pop.Connection, bindModel models.MCIamMappingWorkspaceProjects) (models.MCIamMappingWorkspaceProjects, error) {

	wsPjModel := &models.MCIamMappingWorkspaceProject{}
	for _, obj := range bindModel {
		wsPjModel.ProjectID = obj.ProjectID
		wsPjModel.WorkspaceID = obj.WorkspaceID

		q := tx.Eager().Where("workspace_id = ?", wsPjModel.WorkspaceID)
		q = q.Where("project_id = ?", wsPjModel.ProjectID)
		b, err := q.Exists(wsPjModel)
		if err != nil {
			return nil, errors.New("something query error")
		}

		if b {
			return nil, errors.New("already Exists")
		}

		LogPrintHandler("mapping ws project bind model", wsPjModel)

		//workspace 존재 여부 체크
		wsQuery := models.DB.Where("id = ?", wsPjModel.WorkspaceID)
		existWs, err := wsQuery.Exists(models.MCIamWorkspace{})
		if !existWs {
			cblogger.Error("Workspace not exist, WSID : ", wsPjModel.WorkspaceID)
			return nil, errors.New("Workspace not exist, WSID : " + obj.WorkspaceID)
		}

		//project 존재 여부 체크
		projectQuery := models.DB.Where("id = ?", obj.ProjectID)
		existPj, err := projectQuery.Exists(models.MCIamProject{})
		if !existPj {
			cblogger.Error("Project not exist, PjId : ", wsPjModel.ProjectID)
			return nil, errors.New("Project not exist, PjId : " + wsPjModel.ProjectID)
		}

		err2 := tx.Create(wsPjModel)

		if err2 != nil {
			LogPrintHandler("mapping ws project error", err)

			return nil, err2
		}
	}

	return bindModel, nil
}

func GetMappingProjectByWorkspace(wsId string) (models.MCIamMappingWorkspaceProjects, error) {
	ws := &models.MCIamMappingWorkspaceProjects{}
	parsingWs := models.MCIamMappingWorkspaceProjects{}
	cblogger.Info("wsId : ", wsId)
	wsQuery := models.DB.Eager().Where("workspace_id =?", wsId)
	projects, err := wsQuery.Exists(ws)

	cblogger.Info("projects:", projects)

	if err != nil {
		cblogger.Error(err)
		return models.MCIamMappingWorkspaceProjects{}, err
	}

	if projects {
		err2 := wsQuery.All(ws)

		if err2 != nil {
			cblogger.Error(err)
			return models.MCIamMappingWorkspaceProjects{}, err
		}

		parsingWs = ParserWsProjectByWs(*ws, wsId)
	}

	return parsingWs, nil

}

func MappingWsProjectValidCheck(tx *pop.Connection, wsId string, projectId string) map[string]interface{} {
	ws := &models.MCIamMappingWorkspaceProject{}

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

func ParserWsProjectByWs(bindModels models.MCIamMappingWorkspaceProjects, wsId string) models.MCIamMappingWorkspaceProjects {
	parserWsProjects := models.MCIamMappingWorkspaceProjects{}
	parserWsProject := models.MCIamMappingWorkspaceProject{}
	//projectArray := models.MCIamProjects{}

	cblogger.Info("#### bindmodels ####", bindModels)
	for _, obj := range bindModels {
		cblogger.Info("#### wsuuid ####", obj.WorkspaceID)
		if wsId == obj.WorkspaceID {
			parserWsProject.WorkspaceID = obj.WorkspaceID
			parserWsProject.Workspace = obj.Workspace
			parserWsProject.Project = obj.Project

			if obj.ProjectID != "" {
				//projectArray = append(projectArray, *obj.Project)
				//parserWsProject.Project = projectArray

				parserWsProject.Project = obj.Project
			}
		}

		parserWsProjects = append(parserWsProjects, parserWsProject)
	}

	cblogger.Info("parserWsProject : ", parserWsProject)
	return parserWsProjects
}

// func ParserWsProject(tx *pop.Connection, bindModels []models.MCIamWorkspace) *models.ParserWsProjectMappings {
// 	parserWsProject := []models.ParserWsProjectMapping{}
// 	projectArray := []models.MCIamProject{}

// 	ParserWsProjectByWs

// 	return parserWsProject
// }

func MappingUserRole(tx *pop.Connection, bindModel *models.MCIamMappingWorkspaceUserRole) map[string]interface{} {
	if bindModel.ID != uuid.Nil {
		userRoleModel := &models.MCIamMappingWorkspaceUserRole{}

		roleId := bindModel.RoleName
		userId := bindModel.UserID

		q := tx.Eager().Where("role_name = ?", roleId)
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

func MappingDeleteWsProject(tx *pop.Connection, bindModel *models.MCIamMappingWorkspaceProject) map[string]interface{} {
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
