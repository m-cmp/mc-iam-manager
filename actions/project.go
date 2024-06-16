package actions

import (
	"fmt"
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/handler/mcinframanager"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"

	"github.com/gobuffalo/buffalo"
)

type createProjectRequset struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

type updateProjectRequset struct {
	Description nulls.String `json:"description" db:"description"`
}

func CreateProject(c buffalo.Context) error {
	project := &models.Project{}
	projectReq := &createProjectRequset{}

	err := c.Bind(projectReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*projectReq, project)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	createdProject, err := handler.CreateProject(tx, project)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "project name is duplicated...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	mcInfraCreateNamespaceRequest := &mcinframanager.McInfraCreateNamespaceRequest{}
	err = handler.CopyStruct(*projectReq, mcInfraCreateNamespaceRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	_, err = mcinframanager.McInfraCreateNamespace(mcInfraCreateNamespaceRequest)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "duplicate", "project name is duplicated...")
		err = handler.IsErrorContainsThen(err, "500 Internal Server Error", "tumblebug communicate wrong...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(createdProject))
}

func GetProjectList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	ProjectList, err := handler.GetProjectList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}
	if len(*ProjectList) == 0 {
		return c.Render(http.StatusOK, r.JSON([]map[string]string{}))
	}
	return c.Render(http.StatusOK, r.JSON(ProjectList))
}

func GetProjectByName(c buffalo.Context) error {
	ProjectName := c.Param("projectName")
	tx := c.Value("tx").(*pop.Connection)
	Project, err := handler.GetProjectByName(tx, ProjectName)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(Project))
}

func GetProjectById(c buffalo.Context) error {
	ProjectId := c.Param("projectId")
	tx := c.Value("tx").(*pop.Connection)
	Project, err := handler.GetProjectById(tx, ProjectId)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(Project))
}

func UpdateProjectByName(c buffalo.Context) error {
	projectName := c.Param("projectName")

	project := &models.Project{}
	projectReq := &updateProjectRequset{}

	err := c.Bind(projectReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*projectReq, project)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}
	project.Name = projectName

	tx := c.Value("tx").(*pop.Connection)
	updatedProject, err := handler.UpdateProjectByname(tx, projectName, project)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "the Project you are trying to change is duplicated...")
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the Project you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	mcInfraUpdateNamespaceRequest := &mcinframanager.McInfraUpdateNamespaceRequest{}
	err = handler.CopyStruct(*projectReq, mcInfraUpdateNamespaceRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}
	mcInfraUpdateNamespaceRequest.Name = projectName
	_, err = mcinframanager.McInfraUpdateNamespace(mcInfraUpdateNamespaceRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	return c.Render(http.StatusOK, r.JSON(updatedProject))
}

func UpdateProjectById(c buffalo.Context) error {
	projectId := c.Param("projectId")

	project := &models.Project{}
	projectReq := &updateProjectRequset{}

	err := c.Bind(projectReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	err = handler.CopyStruct(*projectReq, project)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	tx := c.Value("tx").(*pop.Connection)
	projectorg, err := handler.GetProjectById(tx, projectId)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the Project you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	project.Name = projectorg.Name

	updatedProject, err := handler.UpdateProjectById(tx, projectId, project)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "duplicate", "the Project you are trying to change is duplicated...")
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "the Project you are trying to change is not exist...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	mcInfraUpdateNamespaceRequest := &mcinframanager.McInfraUpdateNamespaceRequest{}
	err = handler.CopyStruct(*projectReq, mcInfraUpdateNamespaceRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}
	mcInfraUpdateNamespaceRequest.Name = projectorg.Name
	_, err = mcinframanager.McInfraUpdateNamespace(mcInfraUpdateNamespaceRequest)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	return c.Render(http.StatusOK, r.JSON(updatedProject))
}

func DeleteProjectByName(c buffalo.Context) error {
	projectName := c.Param("projectName")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteProjectByName(tx, projectName)
	fmt.Println("DeleteProjectByName handler done")
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no Project ("+projectName+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	_, err = mcinframanager.McInfraDeleteNamespace(projectName)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": projectName + " is deleted..."}))
}

func DeleteProjectById(c buffalo.Context) error {
	projectId := c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)

	targetProject, err := handler.GetProjectById(tx, projectId)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no Project ("+projectId+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = handler.DeleteProjectById(tx, projectId)
	if err != nil {
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "no Project ("+projectId+") to delete...")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	_, err = mcinframanager.McInfraDeleteNamespace(targetProject.Name)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err.Error()))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": projectId + " is deleted..."}))
}
