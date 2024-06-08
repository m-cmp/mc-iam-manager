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

func CreateProject(c buffalo.Context) error {
	project := &models.MCIamProject{}
	err := c.Bind(project)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatusInternalServerError(err.Error())),
		)
	}
	project.ProjectID = project.Name // TODO : ID, Name 모두 필요한가?

	tx := c.Value("tx").(*pop.Connection)
	createdProject, err := handler.CreateProject(tx, project)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, createdProject)),
	)
}

func GetProjectList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	projectList, err := handler.GetProjectList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, projectList)),
	)
}

func GetProject(c buffalo.Context) error {
	projectId := c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)
	projectList, err := handler.GetProject(tx, projectId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, projectList)),
	)
}

func UpdateProject(c buffalo.Context) error {
	project := &models.MCIamProject{}
	err := c.Bind(project)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	project.ProjectID = c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)
	ProjectList, err := handler.UpdateProject(tx, project)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, ProjectList)),
	)
}

func DeleteProject(c buffalo.Context) error {
	projectId := c.Param("projectId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteProject(tx, projectId)
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
