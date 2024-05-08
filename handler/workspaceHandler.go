package handler

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
	"net/http"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("WorkspaceHandler Resource Test")
	//cblog.SetLevel("info")
	cblog.SetLevel("debug")
}

func CreateWorkspace(tx *pop.Connection, bindModel *iammodels.WorkspaceReq) map[string]interface{} {

	workspace := &models.MCIamWorkspace{
		Name:        bindModel.WorkspaceName,
		Description: bindModel.Description,
	}

	err := tx.Create(workspace)

	if err != nil {
		cblogger.Info("workspace create : ")
		cblogger.Error(err)
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}

	return map[string]interface{}{
		"message":     "success",
		"workspaceId": workspace,
		"status":      http.StatusOK,
	}
}

func UpdateWorkspace(tx *pop.Connection, bindModel iammodels.WorkspaceInfo) map[string]interface{} {
	workspace := &models.MCIamWorkspace{}
	tx.Select().Where("id = ? ", bindModel.WorkspaceId).All(workspace)

	workspace.Name = bindModel.WorkspaceName
	workspace.Description = bindModel.Description

	err := tx.Update(workspace)

	if err != nil {
		cblogger.Info("workspace update : ")
		cblogger.Error(err)
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}

	return map[string]interface{}{
		"message":     "success",
		"workspaceId": workspace,
		"status":      http.StatusOK,
	}
}

//func GetWorkspaceList(tx *pop.Connection) []models.ParserWsProjectMapping {
//	bindModel := []models.MCIamWorkspace{}
//	// projects := &models.MCIamProjects{}
//	// wsProjectMapping := &models.MCIamWsProjectMappings{}
//	err := tx.Eager().All(&bindModel)
//
//	parsingArray := []models.ParserWsProjectMapping{}
//	if err != nil {
//
//	}
//
//	for _, obj := range bindModel {
//		arr := MappingGetProjectByWorkspace(tx, obj.ID.String())
//
//		if arr.WsID != uuid.Nil {
//			parsingArray = append(parsingArray, *arr)
//
//		} else {
//
//			md := models.ParserWsProjectMapping{}
//			ws := models.MCIamWorkspace{}
//			pj := []models.MCIamProject{}
//			ws = obj
//			md.Ws = &ws
//			md.WsID = obj.ID
//			md.Projects = pj
//
//			parsingArray = append(parsingArray, md)
//
//		}
//	}
//
//	return parsingArray
//}

func GetWorkspaceList(userId string) iammodels.WorkspaceInfos {
	var bindModel models.MCIamWorkspaces
	cblogger.Info("userId : " + userId)
	err := models.DB.All(&bindModel)

	if err != nil {
		cblogger.Error(err)
	}

	parsingArray := iammodels.WorkspaceInfos{}

	for _, obj := range bindModel {
		parsingArray = append(parsingArray, iammodels.WorkspaceToWorkspaceInfo(obj, nil))
	}

	return parsingArray
}

func GetWorkspaceListByUserId(userId string) iammodels.WorkspaceInfos {
	wsUserMapping := &models.MCIamWsUserMappings{}
	cblogger.Info("userId : " + userId)
	query := models.DB.Where("user_id=?", userId)

	err := query.All(wsUserMapping)

	parsingArray := iammodels.WorkspaceInfos{}

	cblogger.Info("bindModel :", wsUserMapping)

	if err != nil {
		cblogger.Error(err)
	}

	for _, obj := range *wsUserMapping {
		/**
		1. workspace, user mapping 조회
		2. workspace, projects mapping 조회
		*/
		arr := MappingGetProjectByWorkspace(obj.WsID.String())

		cblogger.Info("arr:", arr)
		if arr.WsID.String() != "00000000-0000-0000-0000-000000000000" {
			info := iammodels.WorkspaceToWorkspaceInfo(*arr.Ws, nil)
			cblogger.Info("Info : ")
			cblogger.Info(info)
			info.ProjectList = iammodels.ProjectsToProjectInfoList(arr.Projects)
			parsingArray = append(parsingArray, info)
		} else {
			parsingArray = append(parsingArray, GetWorkspace(obj.WsID.String()))
		}
	}

	return parsingArray
}

func GetWorkspace(wsId string) iammodels.WorkspaceInfo {
	ws := &models.MCIamWorkspace{}
	err := models.DB.Eager().Find(ws, wsId)
	if err != nil {
		cblogger.Error(err)
	}

	return iammodels.WorkspaceToWorkspaceInfo(*ws, nil)
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
	//만약 삭제가 된다면 mapping table 도 삭제 해야 한다.
	// mapping table 조회
	mws := []models.MCIamWsProjectMapping{}

	err2 := tx.Eager().Where("ws_id =?", wsId).All(&mws)
	if err2 != nil {
		LogPrintHandler("MappingGetProjectByWorkspace", wsId)
	}
	err3 := tx.Destroy(mws)
	if err3 != nil {
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

// Workspace에 할당된 project 조회	GET	/api/ws	/workspace/{workspaceId}/project	AttachedProjectByWorkspace
func AttachedProjectByWorkspace(tx *pop.Connection, wsId string) iammodels.ProjectInfos {
	arr := MappingGetProjectByWorkspace(wsId)

	return iammodels.ProjectsToProjectInfoList(arr.Projects)
}

// Default Workspace 설정/해제 ( setDefault=true/false )	PUT	/api/ws
func AttachedDefaultByWorkspace(tx *pop.Connection) error {
	return nil
}

// Workspace에 Project 할당	POST	/api/ws	/workspace/{workspaceId}/attachproject/{projectId}
//func AttachProjectToWorkspace(tx *pop.Connection, wsId string, pjId string) error {
//	uuidPjId, _ := uuid.FromString(pjId)
//	uuidWsId, _ := uuid.FromString(wsId)
//
//	mapping := models.MCIamWsProjectMapping{ProjectID: uuidPjId, WsID: uuidWsId}
//
//	return nil
//}

// Workspace에 Project 할당 해제	DELELTE	/api/ws	/workspace/{workspaceId}/attachproject/{projectId}
func DeleteProjectFromWorkspace(paramWsId string, paramPjId string, tx *pop.Connection) map[string]interface{} {

	models := &models.MCIamWsProjectMapping{}

	err := tx.Eager().Where("ws_id = ? and project_id =?", paramWsId, paramPjId).First(models)

	if err != nil {
		cblogger.Info(err)
	}

	err2 := tx.Destroy(models)
	if err2 != nil {
		return map[string]interface{}{
			"message": err2,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}
}

// Workspace 사용자 할당 (with Role)	POST	/api/ws	/workspace/{workspaceId}/assigneduser
func AssignUserToWorkspace(tx *pop.Connection) error {
	return nil
}
