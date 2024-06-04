package handler

import (
	"github.com/gobuffalo/nulls"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

func CreateProject(tx *pop.Connection, bindModel *iammodels.ProjectReq) map[string]interface{} {

	project := &models.MCIamProject{
		Name:        bindModel.ProjectName,
		Description: nulls.String{String: bindModel.Description, Valid: true},
	}

	err := tx.Create(project)

	if err != nil {
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"project": project,
		"status":  http.StatusOK,
	}
}

func UpdateProject(tx *pop.Connection, bindModel *iammodels.ProjectInfo) map[string]interface{} {

	project := &models.MCIamProject{
		ID:          uuid.FromStringOrNil(bindModel.ProjectId),
		Name:        bindModel.ProjectName,
		Description: bindModel.Description,
	}

	err := tx.Update(project)

	if err != nil {
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"project": project,
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

func GetProjectListByWorkspaceId(wsId string) (*models.MCIamMappingWorkspaceProjects, error) {
	bindModel := &models.MCIamMappingWorkspaceProjects{}
	query := models.DB.Where("ws_id = " + wsId)
	err := query.All(bindModel)

	if err != nil {
		return nil, err
	}

	return bindModel, err
}

func GetProject(tx *pop.Connection, projectId string) *models.MCIamProject {
	ws := &models.MCIamProject{}

	err := tx.Find(ws, projectId)
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
