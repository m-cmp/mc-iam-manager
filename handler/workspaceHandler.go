package handler

import (
	"github.com/sirupsen/logrus"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
	"net/http"
	"strings"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("WorkspaceHandler Resource Test")
	//cblog.SetLevel("info")
	cblog.SetLevel("debug")
}

func CreateWorkspace(tx *pop.Connection, bindModel *iammodels.WorkspaceReq) map[string]interface{} {
	//cblogger.Info("workspace bindModel : ")
	//cblogger.Info(bindModel)

	workspace := &models.MCIamWorkspace{
		Name:        bindModel.WorkspaceName,
		Description: bindModel.Description,
	}

	cblogger.Info("workspace bindModel : ")
	cblogger.Info(workspace)

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

func GetWorkspaceList(tx *pop.Connection, userId string) iammodels.WorkspaceInfos {
	var bindModel []models.MCIamWorkspace
	cblogger.Info("userId : " + userId)
	var err error

	if len(strings.TrimSpace(userId)) > 0 {
		/**
		To-Do ws_user mapping table crud 다음, 유저 기반으로 검색하도록 작성
		*/
		err = tx.Eager().Where("").All(&bindModel)
	} else {
		err = tx.Eager().All(&bindModel)
	}

	parsingArray := iammodels.WorkspaceInfos{}
	if err != nil {
	}

	for _, obj := range bindModel {
		arr := MappingGetProjectByWorkspace(tx, obj.ID.String())

		if arr.WsID != uuid.Nil {
			info := iammodels.WorkspaceToWorkspaceInfo(*arr.Ws)
			cblogger.Info("Info : ")
			cblogger.Info(info)
			info.ProjectList = iammodels.ProjectsToProjectInfoList(arr.Projects)
			parsingArray = append(parsingArray, info)
		} else {
			parsingArray = append(parsingArray, iammodels.WorkspaceToWorkspaceInfo(obj))
		}

	}

	cblogger.Info("ParsingArray")
	cblogger.Info(parsingArray)
	cblogger.Info("end of ParsingArray")

	return parsingArray
}

func GetWorkspace(tx *pop.Connection, wsId string) iammodels.WorkspaceInfo {
	ws := &models.MCIamWorkspace{}

	err := tx.Eager().Find(ws, wsId)
	if err != nil {

	}

	return iammodels.WorkspaceToWorkspaceInfo(*ws)
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
	arr := MappingGetProjectByWorkspace(tx, wsId)

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

	err := tx.Eager().Where("ws_id = ? and project_id =?", models.WsID, models.ProjectID).All(models)
	cblogger.Info("After Search")
	cblogger.Info(models)
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
