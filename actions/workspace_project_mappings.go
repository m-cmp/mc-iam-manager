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
	var s models.WorkspaceProjectMapping
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	s.WorkspaceID = uuid.FromStringOrNil(req.WorkspaceID)
	for _, projectId := range req.ProjectID {
		s.ProjectID = uuid.FromStringOrNil(projectId)
		tx := c.Value("tx").(*pop.Connection)
		_, err = handler.CreateWPmapping(tx, &s)
		if err != nil {
			log.Println(err)
			err = handler.IsErrorContainsThen(err, "SQLSTATE 23503", "workspace or Project is not exist..")
			err = handler.IsErrorContainsThen(err, "SQLSTATE 23505", "mapping is already exist..")
			return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
		}
	}

	return c.Render(http.StatusOK, r.JSON(nil))
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

func GetWPmappingListByWorkspaceUUID(c buffalo.Context) error {
	var err error
	workspaceUUID := c.Param("workspaceUUID")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetWPmappingListByWorkspaceUUID(tx, uuid.FromStringOrNil(workspaceUUID))
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
	workspaceUUID := c.Param("workspaceUUID")
	projectUUID := c.Param("projectUUID")
	tx := c.Value("tx").(*pop.Connection)
	workspace, err := handler.GetWPmappingByUUID(tx, workspaceUUID, projectUUID)
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
