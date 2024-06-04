package actions

import (
	"github.com/gobuffalo/pop/v6"
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func GetUserRoleType(c buffalo.Context) error {
	roleName := c.Param("roleName")
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

func GetUserRoleTypeList(c buffalo.Context) error {
	resp, err := handler.GetRoles(c, "")
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

func UpdateUserRoleType(c buffalo.Context) error {
	roleBind := &iammodels.RoleTypeReq{}
	if err := c.Bind(roleBind); err != nil {
		handler.LogPrintHandler("role Type bind error", err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}
	tx := c.Value("tx").(*pop.Connection)
	resp, err := handler.UpdateRoleType(tx, *roleBind)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}
	return c.Render(http.StatusOK, r.JSON(resp))
}
func CreateUserRoleType(c buffalo.Context) error {
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

func DeleteUserRoleType(c buffalo.Context) error {
	paramRoleId := c.Param("roleId")

	deleteErr := handler.DeleteRole(c, paramRoleId)
	if deleteErr != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, deleteErr)))
	}
	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "Deleted role successfully")))
}
