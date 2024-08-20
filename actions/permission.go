package actions

import (
	"log"
	"net/http"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

type createPermissionRequset struct {
	Name                string   `json:"name"`
	Desc                string   `json:"desc"`
	PermissionResources []string `json:"resources"`
	PermissionPolicies  []string `json:"policies"`
}

func CreatePermission(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	permissionReq := &createPermissionRequset{}
	if err := c.Bind(permissionReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	resoruces, err := keycloak.KeycloakCreatePermission(accessToken, permissionReq.Name, permissionReq.Desc, permissionReq.PermissionResources, permissionReq.PermissionPolicies)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resoruces))
}

func GetPermissions(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	name := c.Param("name")
	resource := c.Param("resource")
	params := gocloak.GetPermissionParams{
		Resource: &resource,
		Name:     &name,
	}

	resources, err := keycloak.KeycloakGetPermissions(accessToken, params)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(resources))
}

func GetPermission(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	permissionId := c.Param("permissionid")

	resources, err := keycloak.KeycloakGetPermissionDetail(accessToken, permissionId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(resources))
}

func UpdatePermission(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	permissionId := c.Param("permissionid")

	permissionReq := &createPermissionRequset{}
	if err := c.Bind(permissionReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	err := keycloak.KeycloakUpdatePermission(accessToken, permissionId, permissionReq.Name, permissionReq.Desc, permissionReq.PermissionResources, permissionReq.PermissionPolicies)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func DeletePermission(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	permissionId := c.Param("permissionid")

	err := keycloak.KeycloakDeletePermission(accessToken, permissionId)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}
