package actions

import (
	"log"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

func CreateUser(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

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
	accessToken := c.Value("accessToken").(string)

	userid := c.Param("userid")

	users, err := keycloak.KeycloakGetUsers(accessToken, userid)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(users))
}

func DeleteUser(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	userId := c.Param("userId")

	err := keycloak.KeycloakDeleteUser(accessToken, userId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func UpdateUser(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

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
