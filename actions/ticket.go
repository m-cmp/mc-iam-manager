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
	framework := c.Param("framework")

	ticketMenusPermissions, err := keycloak.KeycloakGetAvailableMenus(accessToken, framework)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(ticketMenusPermissions))
}

func GetPermissionTicket(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)
	requestTicketReq := &keycloak.RequestTicket{}
	if err := c.Bind(requestTicketReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}
	ticket, err := keycloak.KeycloakGetPermissionTicket(accessToken, *requestTicketReq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(ticket))
}
