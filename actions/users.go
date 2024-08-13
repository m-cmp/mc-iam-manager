package actions

import (
	"log"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

type createUserRequset struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func CreateUser(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	createUserReq := &createUserRequset{}
	if err := c.Bind(createUserReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	err := keycloak.KeycloakCreateUser(accessToken, createUserReq.Name, createUserReq.Password)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func GetUsers(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	userid := c.Param("userid")

	users, err := keycloak.KeycloakGetUsers(accessToken, userid)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(users))
}

func DeleteUser(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	userId := c.Param("userId")

	err := keycloak.KeycloakDeleteUser(accessToken, userId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}
