package actions

import (
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/pkg/errors"
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
	"net/http"
)

func AssignUserToWorkspace(c buffalo.Context) error {

	wum := &models.MCIamWsUserMapping{}
	if err := c.Bind(wum); err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.MappingWsUser(tx, wum)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingGetWsUserRole(c buffalo.Context) error {
	userId := c.Param("userId")
	// wurm := &models.MCIamWsUserRoleMapping{}
	// if err := c.Bind(wurm); err != nil {

	// }
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.GetWsUserRole(tx, userId)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingUserRole(c buffalo.Context) error {
	urm := &models.MCIamUserRoleMapping{}

	if err := c.Bind(urm); err != nil {

	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingUserRole(tx, urm)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func AttachProjectToWorkspace(c buffalo.Context) error {
	param := &iammodels.WorkspaceProjectMappingReq{}
	c.Bind(param)

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.AttachProjectToWorkspace(tx, iammodels.WsPjMappingreqToModels(*param))

	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingGetProjectByWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")

	resp := handler.MappingGetProjectByWorkspace(paramWsId)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingWsProjectValidCheck(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")
	paramProjectId := c.Param("projectId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingWsProjectValidCheck(tx, paramWsId, paramProjectId)

	return c.Render(http.StatusOK, r.JSON(resp))
}

func MappingDeleteWsProject(c buffalo.Context) error {
	// paramWsId := c.Param("workspaceId")
	// paramProjectId := c.Param("projectId")

	bindModel := &models.MCIamWsProjectMapping{}

	if err := c.Bind(bindModel); err != nil {
		return errors.WithStack(err)
	}

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.MappingDeleteWsProject(tx, bindModel)

	return c.Render(http.StatusOK, r.JSON(resp))

}
