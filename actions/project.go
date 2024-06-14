package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/handler/mcinframanager"
	"mc_iam_manager/models"
	"net/http"
	"strings"

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
			r.JSON(handler.CommonResponseStatusInternalServerError(err.Error())),
		)
	}
	project.ProjectID = project.Name // TODO : ID, Name 모두 필요한가?

	commonRequest := &handler.CommonRequest{
		Request: map[string]string{
			"name":        project.Name,
			"description": project.Description.String,
		},
	}
	commonResponse := mcinframanager.McInfraCreateNamespace(c, commonRequest)
	if commonResponse.Status.StatusCode == 500 && strings.Contains(commonResponse.ResponseData.(map[string]interface{})["message"].(string), "already exists.") {
		commonResponseMsg := commonResponse.ResponseData.(map[string]interface{})["message"].(string)
		log.Println(commonResponseMsg)
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(http.StatusBadRequest, commonResponseMsg)),
		)
	}

	tx := c.Value("tx").(*pop.Connection)
	createdProject, err := handler.CreateProject(tx, project)
	if err != nil {
		log.Println(err)
		commonRequest := &handler.CommonRequest{
			PathParams: map[string]string{
				"nsId": project.ProjectID,
			},
		}
		commonResponse := mcinframanager.McInfraDeleteNamespace(c, commonRequest)
		commonResponseMsg := commonResponse.ResponseData.(map[string]interface{})["message"].(string)
		if commonResponse.Status.StatusCode == 200 {
			return c.Render(http.StatusInternalServerError,
				r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
		} else {
			return c.Render(http.StatusInternalServerError,
				r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, commonResponseMsg+" / "+err.Error())))
		}
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, createdProject)),
	)
}

func GetProjectList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	projectList, err := handler.GetProjectList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, projectList)),
	)
}

func GetProject(c buffalo.Context) error {
	projectId := c.Param("projectId")

	commonRequest := &handler.CommonRequest{
		PathParams: map[string]string{
			"nsId": projectId,
		},
	}

	commonResponse := mcinframanager.McInfraGetNamespace(c, commonRequest)
	if commonResponse.Status.StatusCode == 404 && strings.Contains(commonResponse.ResponseData.(map[string]interface{})["message"].(string), "Not valid namespace") {
		commonResponseMsg := commonResponse.ResponseData.(map[string]interface{})["message"].(string)
		log.Println(commonResponseMsg)
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(http.StatusBadRequest, commonResponseMsg)),
		)
	} else if commonResponse.Status.StatusCode != 200 {
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(commonResponse.Status.StatusCode, commonResponse)),
		)
	}

	tx := c.Value("tx").(*pop.Connection)
	projectList, err := handler.GetProject(tx, projectId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, projectList)),
	)
}

func UpdateProject(c buffalo.Context) error {
	project := &models.MCIamProject{}
	err := c.Bind(project)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatusInternalServerError(err.Error())),
		)
	}
	project.ProjectID = c.Param("projectId")

	commonRequest := &handler.CommonRequest{
		Request: map[string]string{
			"name":        project.Name,
			"description": project.Description.String,
		},
		PathParams: map[string]string{
			"nsId": project.ProjectID,
		},
	}

	commonResponse := mcinframanager.McInfraUpdateNamespace(c, commonRequest)
	if commonResponse.Status.StatusCode == 404 && strings.Contains(commonResponse.ResponseData.(map[string]interface{})["message"].(string), "Not valid namespace") {
		commonResponseMsg := commonResponse.ResponseData.(map[string]interface{})["message"].(string)
		log.Println(commonResponseMsg)
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(http.StatusBadRequest, commonResponseMsg)),
		)
	} else if commonResponse.Status.StatusCode != 200 {
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(commonResponse.Status.StatusCode, commonResponse)),
		)
	}

	tx := c.Value("tx").(*pop.Connection)
	ProjectList, err := handler.UpdateProject(tx, project)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, ProjectList)),
	)
}

func DeleteProject(c buffalo.Context) error {
	projectId := c.Param("projectId")

	commonRequest := &handler.CommonRequest{
		PathParams: map[string]string{
			"nsId": projectId,
		},
	}

	commonResponse := mcinframanager.McInfraDeleteNamespace(c, commonRequest)
	if commonResponse.Status.StatusCode == 404 && strings.Contains(commonResponse.ResponseData.(map[string]interface{})["message"].(string), "Not valid namespace") {
		commonResponseMsg := commonResponse.ResponseData.(map[string]interface{})["message"].(string)
		log.Println(commonResponseMsg)
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(http.StatusBadRequest, commonResponseMsg)),
		)
	} else if commonResponse.Status.StatusCode != 200 {
		return c.Render(http.StatusBadRequest,
			r.JSON(handler.CommonResponseStatus(commonResponse.Status.StatusCode, commonResponse)),
		)
	}

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteProject(tx, projectId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}
	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, nil)),
	)
}

func DebugProject(c buffalo.Context) error {
	a, _ := mcinframanager.McInfraIsExistsNamespace("aaa")
	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, a)),
	)
}
