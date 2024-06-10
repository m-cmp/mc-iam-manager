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

/////////////////////////////
// WorkspaceProjectMapping //
/////////////////////////////

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

// ////////////////////////////
// WorkspaceUserRoleMapping //
// ////////////////////////////

func CreateWorkspaceUserRoleMapping(c buffalo.Context) error {
	mappingWorkspaceUserRole := &models.MCIamMappingWorkspaceUserRole{}
	err := c.Bind(mappingWorkspaceUserRole)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	mappingWorkspaceUserRole.WorkspaceID = c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	createdWorkspaceProjectMapping, err := handler.CreateWorkspaceUserRoleMapping(tx, mappingWorkspaceUserRole)
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

func GetWorkspaceUserRoleMapping(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	createdWorkspaceProjectMapping, err := handler.GetWorkspaceUserRoleMapping(tx)
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

func GetWorkspaceUserRoleMappingByWorkspace(c buffalo.Context) error {

	workspaceId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMappingByWorkspace(tx, workspaceId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, workspaceUserRoleMapping)),
	)
}

func GetWorkspaceUserRoleMappingByWorkspaceUser(c buffalo.Context) error {

	workspaceId := c.Param("workspaceId")
	userId := c.Param("userId")

	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMappingByWorkspaceUser(tx, workspaceId, userId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, workspaceUserRoleMapping)),
	)
}

func GetWorkspaceUserRoleMappingByUser(c buffalo.Context) error {

	userId := c.Param("userId")

	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.GetWorkspaceUserRoleMappingByUser(tx, userId)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, workspaceUserRoleMapping)),
	)
}

func UpdateWorkspaceUserRoleMapping(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	userId := c.Param("userId")

	mappingWorkspaceProject := &models.MCIamMappingWorkspaceUserRole{}
	err := c.Bind(mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatusInternalServerError(err.Error())),
		)
	}

	tx := c.Value("tx").(*pop.Connection)
	workspaceUserRoleMapping, err := handler.UpdateWorkspaceUserRoleMapping(tx, workspaceId, userId, mappingWorkspaceProject)
	if err != nil {
		log.Println(err)
		return c.Render(
			http.StatusInternalServerError,
			r.JSON(iammodels.CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK,
		r.JSON(iammodels.CommonResponseStatus(http.StatusOK, workspaceUserRoleMapping)),
	)
}

func DeleteWorkspaceUserRoleMapping(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	userId := c.Param("userId")

	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceUserRoleMapping(tx, workspaceId, userId)
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

func DeleteWorkspaceUserRoleMappingAll(c buffalo.Context) error {
	workspaceId := c.Param("workspaceId")
	tx := c.Value("tx").(*pop.Connection)
	err := handler.DeleteWorkspaceUserRoleMappingAll(tx, workspaceId)
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
