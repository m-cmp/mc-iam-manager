package actions

import (
	"log"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

func CreateUser(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	createUserReq := &keycloak.CreateUserRequset{}
	if err := c.Bind(createUserReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	err := keycloak.KeycloakCreateUser(accessToken, *createUserReq, createUserReq.Password)
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

func UpdateUser(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	userId := c.Param("userId")

	createUserReq := &keycloak.CreateUserRequset{}
	if err := c.Bind(createUserReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	err := keycloak.KeycloakUpdateUser(accessToken, *createUserReq, userId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}
