package actions

import (
	"log"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

func GetAllPermissions(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	ticketPermissions, err := keycloak.KeycloakGetAvaliablePermissions(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(ticketPermissions))
}

func GetAllAvailableMenus(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	ticketMenusPermissions, err := keycloak.KeycloakGetAvailableMenus(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(ticketMenusPermissions))
}

func GetPermissionTicketByResourceName(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Param("framework")
	operationid := c.Param("operationid")
	ticket, err := keycloak.KeycloakGetTicketByFrameworkResourceName(accessToken, framework, operationid)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(ticket))
}
