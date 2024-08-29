package actions

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

func CreateUser(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	createUserReq := &keycloak.CreateUserRequset{}
	if err := c.Bind(createUserReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: createUserReq.Name, Name: "id"},
		&validators.StringIsPresent{Field: createUserReq.Password, Name: "password"},
		&validators.StringIsPresent{Field: createUserReq.FirstName, Name: "firstName"},
		&validators.StringIsPresent{Field: createUserReq.LastName, Name: "lastName"},
		&validators.StringIsPresent{Field: createUserReq.Email, Name: "email"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": validateErr.Error()}))
	}

	err := keycloak.KeycloakCreateUser(accessToken, *createUserReq, createUserReq.Password)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func ActiveUser(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	activeReq := &keycloak.UserEnableStatusRequest{}
	if err := c.Bind(activeReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: activeReq.UserId, Name: "userId"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": validateErr.Error()}))
	}

	err := keycloak.KeycloakActiveUser(accessToken, activeReq.UserId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func DeactiveUser(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	deactiveReq := &keycloak.UserEnableStatusRequest{}
	if err := c.Bind(deactiveReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: deactiveReq.UserId, Name: "userId"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": validateErr.Error()}))
	}

	err := keycloak.KeycloakDeactiveUser(accessToken, deactiveReq.UserId)
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
