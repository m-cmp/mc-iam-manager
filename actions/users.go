package actions

import (
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// UsersGetUsersList default implementation.
func GetUserList(c buffalo.Context) error {
	userList, err := handler.GetUserList(c)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusOK, r.JSON(err))
	}
	return c.Render(http.StatusOK, r.JSON(userList))
}

func RegistUser(c buffalo.Context) error {
	userInfo := &iammodels.UserReq{}
	c.Bind(userInfo)

	user, err := handler.CreateUser(c, userInfo)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusOK, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(user))
}

func UnRegistUser(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(handler.DeleteUser(c, c.Param("userId"))))
}

func GetUser(c buffalo.Context) error {
	user, err := handler.GetUser(c, c.Param("userId"))

	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusOK, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(user))
}

func UpdateUserProfile(c buffalo.Context) error {
	userInfo := &iammodels.UserInfo{}
	c.Bind(userInfo)

	return c.Render(http.StatusOK, r.JSON(handler.UpdateUser(c, *userInfo)))
}
