package actions

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
	"gopkg.in/yaml.v2"
)

func CreateResources(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	resourcereq := &keycloak.CreateResourceRequestArr{}
	if err := c.Bind(resourcereq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	resoruces, err := keycloak.KeycloakCreateResources(accessToken, *resourcereq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resoruces))
}

type Swagger struct {
	Swagger string                         `yaml:"swagger"`
	Paths   map[string]map[string]struct { // Path 내부 구조체
		Consumes    []string   `yaml:"consumes"`
		Description string     `yaml:"description"`
		OperationID string     `yaml:"operationId"`
		Parameters  []struct { // Parameter 내부 구조체
			Description string   `yaml:"description"`
			In          string   `yaml:"in"`
			Name        string   `yaml:"name"`
			Required    bool     `yaml:"required"`
			Schema      struct { // Schema 내부 구조체
				Ref string `yaml:"$ref"`
			} `yaml:"schema"`
		} `yaml:"parameters"`
		Produces  []string            `yaml:"produces"`
		Responses map[string]struct { // Response 내부 구조체
			Description string `yaml:"description"`
			Schema      struct {
				Ref string `yaml:"$ref"`
			} `yaml:"schema"`
		} `yaml:"responses"`
		Summary string   `yaml:"summary"`
		Tags    []string `yaml:"tags"`
	} `yaml:"paths"`
	SecurityDef map[string]struct { // SecurityDefinition 내부 구조체
		Type        string `yaml:"type"`
		Description string `yaml:"description,omitempty"`
		In          string `yaml:"in,omitempty"`
		Name        string `yaml:"name,omitempty"`
	} `yaml:"securityDefinitions"`
}

func CreateResourcesBySwagger(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	framework := c.Param("framework")

	f, err := c.File("swagger")
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	fileContent, err := io.ReadAll(f.File)
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	var swagger Swagger
	err = yaml.Unmarshal(fileContent, &swagger)
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	resourcereq := keycloak.CreateResourceRequestArr{}
	for path, pathData := range swagger.Paths {
		for method, item := range pathData {
			resource := keycloak.CreateResourceRequest{
				Framework:   framework,
				URI:         path,
				Method:      method,
				OperationId: item.OperationID,
			}
			resourcereq = append(resourcereq, resource)
		}
	}

	resoruces, err := keycloak.KeycloakCreateResources(accessToken, resourcereq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resoruces))
}

func GetResources(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	uri := c.Param("uri")
	name := c.Param("operationId")
	params := gocloak.GetResourceParams{
		URI:  &uri,
		Name: &name,
	}

	resources, err := keycloak.KeycloakGetResources(accessToken, params)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(resources))
}

func UpdateResource(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	resourceid := c.Param("resourceid")
	resourcereq := &keycloak.CreateResourceRequest{}
	if err := c.Bind(resourcereq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	err := keycloak.KeycloakUpdateResources(accessToken, resourceid, *resourcereq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func DeleteResource(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	resourceid := c.Param("resourceid")

	err := keycloak.KeycloakDeleteResources(accessToken, resourceid)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}
