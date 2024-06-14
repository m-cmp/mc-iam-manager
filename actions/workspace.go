package actions

import (
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"

	"github.com/gobuffalo/buffalo"
)

func CreateWorkspace(c buffalo.Context) error {
	workspace := &models.MCIamWorkspace{}
	err := c.Bind(workspace)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatusInternalServerError(err.Error())),
		)
	}
	workspace.WorkspaceID = workspace.Name // TODO : ID, Name 모두 필요한가?

	tx := c.Value("tx").(*pop.Connection)
	createdWorkspace, err := handler.CreateWorkspace(tx, workspace)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, createdWorkspace)),
	)
}

func GetWorkspaceList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	workspaceList, err := handler.GetWorkspaceList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, workspaceList)),
	)
}

func GetWorkspace(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	workspaceList, err := handler.GetWorkspace(tx, workspaceId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, workspaceList)),
	)
}

func UpdateWorkspace(c buffalo.Context) error {
	workspace := &models.MCIamWorkspace{}
	err := c.Bind(workspace)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	workspace.WorkspaceID = c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	workspaceList, err := handler.UpdateWorkspace(tx, workspace)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(handler.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(handler.CommonResponseStatus(http.StatusOK, workspaceList)),
	)
}

func DeleteWorkspace(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspace(tx, workspaceId)
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
