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

func GetWorkspaceList(tx *pop.Connection) []models.ParserWsProjectMapping {
	bindModel := []models.MCIamWorkspace{}
	// projects := &models.MCIamProjects{}
	// wsProjectMapping := &models.MCIamWsProjectMappings{}
	err := tx.Eager().All(&bindModel)
	parsingArray := []models.ParserWsProjectMapping{}
	if err != nil {

	}

	for _, obj := range bindModel {
		arr := MappingGetProjectByWorkspace(tx, obj.ID.String())

		if arr.WsID != uuid.Nil {
			parsingArray = append(parsingArray, *arr)
		} else {
			md := &models.ParserWsProjectMapping{}
			pj := []models.MCIamProject{}
			md.Ws = &obj
			md.WsID = obj.ID
			md.Projects = pj
			parsingArray = append(parsingArray, *md)
			LogPrintHandler("print object", obj)
		}
	}

	return parsingArray
}

func GetWorkspace(tx *pop.Connection, wsId string) *models.MCIamWorkspace {
	ws := &models.MCIamWorkspace{}

	err := tx.Eager().Find(ws, wsId)
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
