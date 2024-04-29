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
		return c.Render(http.StatusOK, r.JSON(err))
	}
	return c.Render(http.StatusOK, r.JSON(userList))
}

func CreateUserGroup(c buffalo.Context) error {
	userGroupInfo := &iammodels.UserGroupReq{}
	c.Bind(userGroupInfo)

	userGroup, err := handler.CreateUserGroup(c, userGroupInfo)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusOK, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(userGroup))
}

func DeleteUserGroup(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(handler.DeleteUserGroup(c, c.Param("groupId"))))
}

func GetUserGroup(c buffalo.Context) error {
	user, err := handler.GetUserGroup(c, c.Param("groupId"))

	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusOK, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(user))
}

func UpdateUserGroup(c buffalo.Context) error {
	userGroupInfo := &iammodels.UserGroupInfo{}
	c.Bind(userGroupInfo)

	return c.Render(http.StatusOK, r.JSON(handler.UpdateUserGroup(c, *userGroupInfo)))
}
