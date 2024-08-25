package actions

import (
	"log"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

func GetAllPermissionTicket(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	ticket, err := keycloak.KeycloakGetAvaliablePermission(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]interface{}{"status": ticket}))
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
	return c.Render(http.StatusOK, r.JSON(map[string]interface{}{"status": ticket}))
}

func GetAvailableMenus(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	ticket, err := keycloak.KeycloakGetAvailableMenus(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]interface{}{"menus": ticket}))
}

func TicketValidate(c buffalo.Context) error {
	// accessToken := c.Value("accessToken").(string)

	// resp, err := mcinframanager.McInfraListAllNamespaces()
	// if err != nil {
	// 	log.Println(err)
	// 	return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	// }
	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "OK"}))
}
