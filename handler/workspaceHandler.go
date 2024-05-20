package handler

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("WorkspaceHandler Resource Test")
	//cblog.SetLevel("info")
	cblog.SetLevel("debug")
}

func CreateWorkspace(tx *pop.Connection, bindModel *iammodels.WorkspaceReq) (iammodels.WorkspaceInfo, error) {

	workspace := &models.MCIamWorkspace{
		Name:        bindModel.WorkspaceName,
		Description: nulls.String{String: bindModel.Description, Valid: true},
	}

	err := tx.Create(workspace)

	if err != nil {
		cblogger.Info("workspace create : ")
		cblogger.Error(err)
		return iammodels.WorkspaceInfo{}, err
	}

	return iammodels.WorkspaceToWorkspaceInfo(*workspace, nil), nil
}

func UpdateWorkspace(tx *pop.Connection, bindModel iammodels.WorkspaceInfo) (iammodels.WorkspaceInfo, error) {
	workspace := &models.MCIamWorkspace{}
	tx.Select().Where("id = ? ", bindModel.WorkspaceId).All(workspace)

	workspace.Name = bindModel.WorkspaceName
	workspace.Description = bindModel.Description

	err := tx.Update(workspace)

	if err != nil {
		cblogger.Info("workspace update : ")
		cblogger.Error(err)
		return iammodels.WorkspaceInfo{}, err
	}

	return iammodels.WorkspaceToWorkspaceInfo(*workspace, nil), nil
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
//		arr := GetMappingProjectByWorkspace(tx, obj.ID.String())
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

func GetWorkspaceList(userId string) (iammodels.WorkspaceInfos, error) {
	var bindModel models.MCIamWorkspaces
	cblogger.Info("userId : " + userId)
	err := models.DB.All(&bindModel)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	parsingArray := iammodels.WorkspaceInfos{}

	for _, obj := range bindModel {
		parsingArray = append(parsingArray, iammodels.WorkspaceToWorkspaceInfo(obj, nil))
	}

	return parsingArray, nil
}

func GetWorkspaceListByUserId(userId string) (iammodels.WorkspaceInfos, error) {
	wsUserMapping := &models.MCIamMappingWorkspaceUserRoles{}
	cblogger.Info("userId : " + userId)
	query := models.DB.Where("user_id=?", userId)

	err := query.All(wsUserMapping)

	parsingArray := iammodels.WorkspaceInfos{}

	cblogger.Info("bindModel :", wsUserMapping)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	for _, obj := range *wsUserMapping {
		/**
		1. workspace, user mapping 조회
		2. workspace, projects mapping 조회
		*/
		arr, err2 := GetMappingProjectByWorkspace(obj.WorkspaceID)

		if err2 != nil {
			cblogger.Error(err2)
		}
		cblogger.Info("arr:", arr)

		for _, mapper := range arr {
			if mapper.WorkspaceID != "00000000-0000-0000-0000-000000000000" {
				info := iammodels.WorkspaceToWorkspaceInfo(*mapper.Workspace, nil)
				cblogger.Info("Info : ")
				cblogger.Info(info)
				info.ProjectList = append(info.ProjectList, iammodels.ProjectToProjectInfo(*mapper.Project))
				parsingArray = append(parsingArray, info)
			} else {
				workspace, _ := GetWorkspace(obj.WorkspaceID)
				parsingArray = append(parsingArray, workspace)
			}
		}
	}

	return parsingArray, nil
}

func GetWorkspace(wsId string) (iammodels.WorkspaceInfo, error) {
	ws := &models.MCIamWorkspace{}
	err := models.DB.Eager().Find(ws, wsId)
	if err != nil {
		cblogger.Error(err)
		return iammodels.WorkspaceInfo{}, err
	}

	return iammodels.WorkspaceToWorkspaceInfo(*ws, nil), nil
}

func DeleteWorkspace(tx *pop.Connection, wsId string) error {
	ws := &models.MCIamWorkspace{}
	wsUuid, _ := uuid.FromString(wsId)
	ws.ID = wsUuid

	err := tx.Destroy(ws)
	if err != nil {
		return err
	}
	//만약 삭제가 된다면 mapping table 도 삭제 해야 한다.
	// mapping table 조회
	var mws models.MCIamMappingWorkspaceProjects

	err2 := tx.Eager().Where("ws_id =?", wsId).All(&mws)
	if err2 != nil {
		LogPrintHandler("GetMappingProjectByWorkspace", wsId)
	}
	err3 := tx.Destroy(mws)
	if err3 != nil {
		return err3
	}
	return nil
}

// Workspace에 할당된 project 조회	GET	/api/ws	/workspace/{workspaceId}/project	AttachedProjectByWorkspace
func AttachedProjectByWorkspace(wsId string) (iammodels.ProjectInfos, error) {
	arr, err := GetMappingProjectByWorkspace(wsId)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	var projects iammodels.ProjectInfos
	for _, mapper := range arr {
		projects = append(projects, iammodels.ProjectToProjectInfo(*mapper.Project))
	}

	return projects, nil
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
func DeleteProjectFromWorkspace(paramWsId string, paramPjId string, tx *pop.Connection) error {

	model := &models.MCIamMappingWorkspaceProject{}

	err := tx.Eager().Where("ws_id = ? and project_id =?", paramWsId, paramPjId).First(model)

	if err != nil {
		cblogger.Info(err)
		return err
	}

	err2 := tx.Destroy(model)
	if err2 != nil {
		cblogger.Info(err2)
		return err2
	}
	return nil
}
