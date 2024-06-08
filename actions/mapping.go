package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"

	"github.com/gobuffalo/buffalo"
)

func CreateWorkspaceProjectMapping(c buffalo.Context) error {
	mappingWorkspaceProject := &models.MCIamMappingWorkspaceProjectRequest{}
	err := c.Bind(mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	mappingWorkspaceProject.WorkspaceID = c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	createdWorkspaceProjectMapping, err := handler.CreateWorkspaceProjectMapping(tx, mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, createdWorkspaceProjectMapping)),
	)
}

func GetWorkspaceProjectMappingList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	WorkspaceProjectMappingList, err := handler.GetWorkspaceProjectMappingList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, WorkspaceProjectMappingList)),
	)
}

func GetWorkspaceProjectMappingByWorkspace(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	WorkspaceProjectMappingList, err := handler.GetWorkspaceProjectMapping(tx, workspaceId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, WorkspaceProjectMappingList)),
	)
}

func UpdateWorkspaceProjectMapping(c buffalo.Context) error {
	mappingWorkspaceProject := &models.MCIamMappingWorkspaceProjectRequest{}
	err := c.Bind(mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	mappingWorkspaceProject.WorkspaceID = c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	createdWorkspaceProjectMapping, err := handler.UpdateWorkspaceProjectMapping(tx, mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, createdWorkspaceProjectMapping)),
	)
}

func DeleteWorkspaceProjectMapping(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	projectId := c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceProjectMapping(tx, workspaceId, projectId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, nil)),
	)
}

func DeleteWorkspaceProjectMappingAllByWorkspace(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceProjectMappingAllByWorkspace(tx, workspaceId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, nil)),
	)
}

func DeleteWorkspaceProjectMappingByProject(c buffalo.Context) error {
	workspaceId := c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceProjectMappingAllByWorkspace(tx, workspaceId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, nil)),
	)
}
