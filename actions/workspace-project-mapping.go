package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

func CreateWorkspaceProjectMappingByName(c buffalo.Context) error {
	mappingWorkspaceProjectsRequest := &models.MappingWorkspaceProjectsNameRequest{}
	workspaceName := c.Param("workspaceName")

	err := c.Bind(mappingWorkspaceProjectsRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err.Error()))
	}
	mappingWorkspaceProjectsRequest.WorkspaceName = workspaceName

	tx := c.Value("tx").(*pop.Connection)
	createWorkspaceProjectMapping, err := handler.CreateWorkspaceProjectMappingByWorkspaceAndProjectsName(tx, mappingWorkspaceProjectsRequest)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace or project is not exist..")
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "there is duplicated mapping..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(createWorkspaceProjectMapping))
}

func CreateWorkspaceProjectMappingById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	mappingWorkspaceProjectsRequest := &models.MappingWorkspaceProjectsIdRequest{}
	err := c.Bind(mappingWorkspaceProjectsRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err.Error()))
	}
	mappingWorkspaceProjectsRequest.WorkspaceId = workspaceId

	tx := c.Value("tx").(*pop.Connection)
	createWorkspaceProjectMapping, err := handler.CreateWorkspaceProjectMappingByWorkspaceAndProjectsId(tx, mappingWorkspaceProjectsRequest)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "workspace or project is not exist..")
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "there is duplicated mapping..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(createWorkspaceProjectMapping))
}

func GetWorkspaceProjectMappingListByWorkspace(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	mappingList, err := handler.GetWorkspaceProjectMappingListByWorkspace(tx)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "is not exist..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(mappingList))
}

func GetWorkspaceProjectMappingByWorkspaceName(c buffalo.Context) error {
	workspaceName := c.Param("workspaceName")

	tx := c.Value("tx").(*pop.Connection)
	mappingList, err := handler.GetWorkspaceProjectMappingByWorkspaceName(tx, workspaceName)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", workspaceName+" mapping is not exist..")

		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(mappingList))
}

func GetWorkspaceProjectMappingByWorkspaceId(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	mappingList, err := handler.GetWorkspaceProjectMappingByWorkspaceId(tx, workspaceId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", workspaceId+" mapping is not exist..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(mappingList))
}

func DeleteWorkspaceProjectMappingById(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	projectId := c.Param("projectId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceProjectMappingById(tx, workspaceId, projectId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", workspaceId+" - "+projectId+" mapping is not exist..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": workspaceId + " and " + projectId + " mapping delete succes.."}))
}

func DeleteWorkspaceProjectMappingByName(c buffalo.Context) error {
	mappingWorkspaceProject := &models.MappingWorkspaceProjectsDeleteNameRequest{}
	workspaceName := c.Param("workspaceName")
	projectName := c.Param("projectName")
	mappingWorkspaceProject.WorkspaceName = workspaceName
	mappingWorkspaceProject.ProjectName = projectName
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceProjectMappingByName(tx, mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", workspaceName+" - "+projectName+" mapping is not exist..")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": workspaceName + " - " + projectName + " mapping delete succes.."}))
}
