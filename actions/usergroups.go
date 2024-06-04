package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// UsersGetUsersList default implementation.
func GetUserGroupList(c buffalo.Context) error {
	userList, err := handler.GetUserGroupList(c)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, userList)))
}

func CreateUserGroup(c buffalo.Context) error {
	userGroupInfo := &iammodels.UserGroupReq{}
	err := c.Bind(userGroupInfo)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	userGroup, err := handler.CreateUserGroup(c, userGroupInfo)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, userGroup)))
}

func DeleteUserGroup(c buffalo.Context) error {
	err := handler.DeleteUserGroup(c, c.Param("groupId"))
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "Deleted user group Successfully")))
}

func GetUserGroup(c buffalo.Context) error {
	user, err := handler.GetUserGroup(c, c.Param("groupId"))

	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, user)))
}

func UpdateUserGroup(c buffalo.Context) error {
	userGroupInfo := &iammodels.UserGroupInfo{}
	err := c.Bind(userGroupInfo)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}
	cblogger.Info("GroupInfo : ", userGroupInfo)
	userGroup, updateErr := handler.UpdateUserGroup(c, *userGroupInfo)

	if updateErr != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, updateErr)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, userGroup)))
}
