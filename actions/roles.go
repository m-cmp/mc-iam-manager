package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func GetUserRole(c buffalo.Context) error {
	roleName := c.Param("roleId")
	resp, err := handler.GetRole(c, roleName)

	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(resp))
}

// func GetRoleByUser(c buffalo.Context) error {
// 	tx := c.Value("tx").(*pop.Connection)
// 	userId := c.Param("userId")

// 	resp := handler.GetRoleByUser(tx, userId)
// 	return c.Render(http.StatusOK,r.JSON(resp))
// }

func GetUserRoleList(c buffalo.Context) error {
	resp, err := handler.GetRoles(c, "")
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

func UpdateUserRole(c buffalo.Context) error {

	roleBind := &iammodels.RoleReq{}
	if err := c.Bind(roleBind); err != nil {
		handler.LogPrintHandler("role bind error", err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	resp, err := handler.UpdateRole(c, *roleBind)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}
	return c.Render(http.StatusOK, r.JSON(resp))
}
func CreateUserRole(c buffalo.Context) error {
	roleReq := &iammodels.RoleReq{}
	//roleBind := &models.MCIamRole{}
	if err := c.Bind(roleReq); err != nil {
		handler.LogPrintHandler("role bind error", err)

		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	handler.LogPrintHandler("role bind", roleReq)

	resp, err := handler.CreateRole(c, roleReq)

	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusAccepted, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

func DeleteUserRole(c buffalo.Context) error {
	paramRoleId := c.Param("roleId")

	deleteErr := handler.DeleteRole(c, paramRoleId)
	if deleteErr != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, deleteErr)))
	}
	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "Deleted role successfully")))
}

// POST	/api/auth	/usergroup/{groupId}/assignuser	AssignUserToUserGroup
func AssignUserToUserGroup(c buffalo.Context) error {
	userRoleInfo := &iammodels.UserRoleInfo{}
	err := c.Bind(userRoleInfo)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "")))
}

// UPDATE	/api/auth	/usergroup/{groupId}/unassign	UnassignUserFromUserGroup
func UnassignUserFromUserGroup(c buffalo.Context) error {

	return nil
}
