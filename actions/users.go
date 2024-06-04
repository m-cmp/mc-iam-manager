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
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}
	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, userList)))
}

func RegistUser(c buffalo.Context) error {
	userInfo := &iammodels.UserReq{}
	err := c.Bind(userInfo)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	cblogger.Info(userInfo)

	user, createErr := handler.CreateUser(c, userInfo)
	if createErr != nil {
		cblogger.Error(createErr)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, createErr)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, user)))
}

func UnRegistUser(c buffalo.Context) error {
	userParam := c.Param("userId")
	err := handler.DeleteUser(c, userParam)

	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "delete success")))
}

func GetUser(c buffalo.Context) error {
	userParam := c.Param("userId")
	user, err := handler.GetUser(c, userParam)

	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, user)))
}

func UpdateUserProfile(c buffalo.Context) error {
	userInfo := &iammodels.UserInfo{}
	err := c.Bind(userInfo)
	if err != nil {
		cblogger.Error(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}
	updateUser, updateErr := handler.UpdateUser(c, *userInfo)

	if updateErr != nil {
		cblogger.Error(updateErr)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err)))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, updateUser)))
}
