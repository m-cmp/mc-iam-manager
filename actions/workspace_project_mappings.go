package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type createWPmappingRequest struct {
	WorkspaceID string   `json:"workspaceId"`
	ProjectID   []string `json:"projectIds"`
}

func CreateWPmappings(c buffalo.Context) error {
	var req createWPmappingRequest
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	for _, projectId := range req.ProjectID {
		var s models.WorkspaceProjectMapping
		s.WorkspaceID = uuid.FromStringOrNil(req.WorkspaceID)
		s.ProjectID = uuid.FromStringOrNil(projectId)

		_, err = handler.CreateWPmapping(tx, &s)
		if err != nil {
			log.Println(err)
			err = handler.IsErrorContainsThen(err, "SQLSTATE 23503", "workspace or Project is not exist..")
			err = handler.IsErrorContainsThen(err, "SQLSTATE 23505", "mapping is already exist..")
			return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
		}
	}
	resp, err := handler.GetWPmappingListByWorkspaceId(tx, uuid.FromStringOrNil(req.WorkspaceID))
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resp))
}

func GetWPmappingListOrderbyWorkspace(c buffalo.Context) error {
	var err error
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetWPmappingListOrderbyWorkspace(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetWPmappingListByWorkspaceId(c buffalo.Context) error {
	var err error
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetWPmappingListByWorkspaceId(tx, uuid.FromStringOrNil(workspaceId))
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

type updateWPmappingRequest struct {
	WorkspaceID string   `json:"workspaceId"`
	ProjectIDs  []string `json:"projectIds"`
}

func UpdateWPmappings(c buffalo.Context) error {
	var req updateWPmappingRequest
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.UpdateWPmappings(tx, req.WorkspaceID, req.ProjectIDs)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resp))
}

func DeleteWPmapping(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	projectId := c.Param("projectId")
	tx := c.Value("tx").(*pop.Connection)
	workspace, err := handler.GetWPmappingById(tx, workspaceId, projectId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = handler.DeleteWPmapping(tx, workspace)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "done"}))
}
