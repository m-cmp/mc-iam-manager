package actions

import (
	"io"
	"log"
	"net/http"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
	"gopkg.in/yaml.v2"
)

func CreateResources(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

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
	accessToken := c.Value("accessToken").(string)

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

// web:menu:<Id>:<DisplayName>:<ParentMenuId>:<Priority>:<IsAction>
// web:menu:settings:settings category::0:false
type Menu struct {
	Id           string `json:"id"` // for routing
	ParentMenuId string `json:"parentmenuid"`
	DisplayName  string `json:"displayname"` // for display
	IsAction     string `json:"isaction"`    // maybe need type assertion..?
	Priority     string `json:"priority"`
	Menus        []Menu `json:"menus"`
}

func CreateMenuResourcesByYaml(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Param("framework")

	f, err := c.File("yaml")
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	fileContent, err := io.ReadAll(f.File)
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	menus := &Menu{}
	err = yaml.Unmarshal(fileContent, &menus)
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	resourcereq := keycloak.CreateMenuResourceRequestArr{}
	for _, menu := range menus.Menus {
		resource := keycloak.CreateMenuResourceRequest{
			Framework:    framework,
			Id:           menu.Id,
			ParentMenuId: menu.ParentMenuId,
			DisplayName:  menu.DisplayName,
			IsAction:     menu.IsAction,
			Priority:     menu.Priority,
		}
		resourcereq = append(resourcereq, resource)
	}

	resoruces, err := keycloak.KeycloakCreateMenuResources(accessToken, resourcereq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resoruces))
}

func GetResources(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	uri := c.Param("uri")
	name := c.Param("operationId")
	params := gocloak.GetResourceParams{
		URI:  &uri,
		Name: gocloak.StringP(":res:" + name),
	}

	resources, err := keycloak.KeycloakGetResources(accessToken, params)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(resources))
}

func GetMenuResources(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)
	framework := c.Param("framework")

	params := gocloak.GetResourceParams{
		Name: gocloak.StringP(framework + ":menu:"),
	}

	resources, err := keycloak.KeycloakGetResources(accessToken, params)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(resources))
}

func UpdateResource(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

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
	accessToken := c.Value("accessToken").(string)

	resourceid := c.Param("resourceid")

	err := keycloak.KeycloakDeleteResource(accessToken, resourceid)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}

func ResetResource(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	resources, err := keycloak.KeycloakGetResources(accessToken, gocloak.GetResourceParams{})
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}
	for _, resource := range resources {
		err := keycloak.KeycloakDeleteResource(accessToken, *resource.ID)
		if err != nil {
			log.Println(err)
			return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
		}
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
}
