package actions

import (
	"fmt"
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

	f, err := c.File("file")
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

type ApiYaml struct {
	CLISpecVersion string `yaml:"cliSpecVersion"`
	Services       map[string]struct {
		BaseURL string `yaml:"baseurl"`
		Auth    struct {
			Type     string `yaml:"type"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"auth"`
	} `yaml:"services"`
	ServiceActions map[string]map[string]struct {
		Method       string `yaml:"method"`
		ResourcePath string `yaml:"resourcePath"`
		Description  string `yaml:"description"`
	} `yaml:"serviceActions"`
}

func CreateApiResourcesByApiYaml(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	reqframework := c.Param("framework")

	f, err := c.File("file")
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	fileContent, err := io.ReadAll(f.File)
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	var apiYaml ApiYaml
	err = yaml.Unmarshal(fileContent, &apiYaml)
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	resourcereq := keycloak.CreateResourceRequestArr{}
	for framework, resources := range apiYaml.ServiceActions {
		for operationId, resource := range resources {
			if framework == reqframework {
				resourcereq = append(resourcereq, keycloak.CreateResourceRequest{
					Framework:   framework,
					URI:         resource.ResourcePath,
					Method:      resource.Method,
					OperationId: operationId,
				})
			} else if reqframework == "all" {
				resourcereq = append(resourcereq, keycloak.CreateResourceRequest{
					Framework:   framework,
					URI:         resource.ResourcePath,
					Method:      resource.Method,
					OperationId: operationId,
				})
			}
		}
	}

	resoruces, err := keycloak.KeycloakCreateResources(accessToken, resourcereq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(resoruces))
}

type Menu struct {
	Id          string `json:"id"` // for routing
	ParentId    string `json:"parentid"`
	DisplayName string `json:"displayname"` // for display
	ResType     string `json:"restype"`
	IsAction    string `json:"isaction"` // maybe need type assertion
	Priority    string `json:"priority"`
	MenuNumber  string `json:"menunumber"`
	Menus       []Menu `json:"menus"`
}

func CreateWebResourceResourcesByMenuYaml(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Param("framework")

	f, err := c.File("file")
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
			Framework:   framework,
			Id:          menu.Id,
			ParentId:    menu.ParentId,
			DisplayName: menu.DisplayName,
			ResType:     menu.ResType,
			IsAction:    menu.IsAction,
			MenuNumber:  menu.MenuNumber,
			Priority:    menu.Priority,
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
	framework := c.Param("framework")
	params := gocloak.GetResourceParams{
		URI:  &uri,
		Name: gocloak.StringP(framework + ":res:" + name),
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

func GetMenuResourcesByType(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)
	framework := c.Param("framework")
	resourceType := c.Param("resourceType")

	params := gocloak.GetResourceParams{
		Name: gocloak.StringP(framework + ":" + resourceType + ":"),
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

func ResetMenuResource(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Request().URL.Query().Get("framework")
	var errs []string
	var resources []*gocloak.ResourceRepresentation
	for {
		resourcesFetch, err := keycloak.KeycloakGetResources(accessToken, gocloak.GetResourceParams{
			Name: gocloak.StringP(framework + ":menu:"),
		})
		if err != nil {
			log.Println(err)
			return c.Render(http.StatusBadRequest, r.JSON(err))
		}
		resources = append(resources, resourcesFetch...)
		for _, resource := range resourcesFetch {
			err := keycloak.KeycloakDeleteResource(accessToken, *resource.ID)
			if err != nil {
				log.Println(err)
				errs = append(errs, err.Error())
			}
		}
		if len(resourcesFetch) < 100 {
			break
		}
	}

	msg := fmt.Sprintf("delete Success %d / %d", len(resources)-len(errs), len(resources))

	if len(errs) > 0 {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"status": msg, "errors": strings.Join(errs, "; ")}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": msg}))
}

func ResetResource(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Request().URL.Query().Get("framework")
	var errs []string
	var resources []*gocloak.ResourceRepresentation
	for {
		resourcesFetch, err := keycloak.KeycloakGetResources(accessToken, gocloak.GetResourceParams{
			Name: gocloak.StringP(framework + ":res:"),
		})
		if err != nil {
			log.Println(err)
			return c.Render(http.StatusBadRequest, r.JSON(err))
		}
		resources = append(resources, resourcesFetch...)
		for _, resource := range resourcesFetch {
			err := keycloak.KeycloakDeleteResource(accessToken, *resource.ID)
			if err != nil {
				log.Println(err)
				errs = append(errs, err.Error())
			}
		}
		if len(resourcesFetch) < 100 {
			break
		}
	}

	msg := fmt.Sprintf("delete Success %d / %d", len(resources)-len(errs), len(resources))

	if len(errs) > 0 {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"status": msg, "errors": strings.Join(errs, "; ")}))
	}
	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": msg}))
}
