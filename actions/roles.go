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
		return c.Render(http.StatusInternalServerError, r.JSON(err))
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
	resp := handler.GetRoles(c, "")
	return c.Render(http.StatusOK, r.JSON(resp))
}

func UpdateUserRole(c buffalo.Context) error {

	roleBind := &iammodels.RoleReq{}
	if err := c.Bind(roleBind); err != nil {
		handler.LogPrintHandler("role bind error", err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	resp := handler.UpdateRole(c, *roleBind)

	return c.Render(http.StatusOK, r.JSON(resp))
}
func CreateUserRole(c buffalo.Context) error {
	roleReq := &iammodels.RoleReq{}
	//roleBind := &models.MCIamRole{}
	if err := c.Bind(roleReq); err != nil {
		handler.LogPrintHandler("role bind error", err)

		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	handler.LogPrintHandler("role bind", roleReq)

	resp := handler.CreateRole(c, roleReq)

	return c.Render(http.StatusAccepted, r.JSON(resp))
}

func DeleteUserRole(c buffalo.Context) error {
	paramRoleId := c.Param("roleId")

	resp := handler.DeleteRole(c, paramRoleId)
	return c.Render(http.StatusOK, r.JSON(resp))
}

// POST	/api/auth	/usergroup/{groupId}/assignuser	AssignUserToUserGroup
func AssignUserToUserGroup(c buffalo.Context) error {
	userRoleInfo := &iammodels.UserRoleInfo{}
	c.Bind(userRoleInfo)

	return nil
}

// UPDATE	/api/auth	/usergroup/{groupId}/unassign	UnassignUserFromUserGroup
func UnassignUserFromUserGroup(c buffalo.Context) error {

	return nil
}
