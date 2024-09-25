package actions

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/m-cmp/mc-iam-manager/handler"
	"github.com/m-cmp/mc-iam-manager/handler/mcinframanager"
	"github.com/m-cmp/mc-iam-manager/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
)

func SyncProjectListWithMcInfra(c buffalo.Context) error {
	resp, err := mcinframanager.McInfraListAllNamespaces()
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	var nsList map[string]interface{}
	err = json.Unmarshal(resp, &nsList)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	tx := c.Value("tx").(*pop.Connection)

	var errs []error
	if nsArray, ok := nsList["ns"].([]interface{}); ok {
		for _, item := range nsArray {
			if nsItem, ok := item.(map[string]interface{}); ok {
				var s models.Project
				s.NsID = nsItem["id"].(string)
				s.Name = nsItem["name"].(string)
				if description, ok := nsItem["description"].(nulls.String); ok {
					s.Description = description
				}
				project, _ := handler.IsExistProjectByNsId(tx, s.NsID)

				if project == nil {
					_, err := handler.CreateProject(tx, &s)
					if err != nil {
						log.Println(err)
						errs = append(errs, err)
					}
				}
			}
		}
	}

	projectList, err := handler.GetProjectList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	if len(errs) > 0 {
		log.Println(errs)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]interface{}{"projectList": projectList, "errors": errs}))
	}

	return c.Render(http.StatusOK, r.JSON(projectList))
}

// func SyncRoleListWithKeycloak(c buffalo.Context) error {
// 	token := c.Value("accessToken").(string)

// 	roles, err := keycloak.KeycloakGetRoles(token)
// 	if err != nil {
// 		log.Println(err)
// 		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
// 	}

// 	tx := c.Value("tx").(*pop.Connection)
// 	roleRes, err := handler.CreateRole(tx, &s)
// 	if err != nil {
// 		log.Println(err)
// 		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "Role is already exist..")
// 		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
// 	}

// 	return c.Render(http.StatusOK, r.JSON(roles))

// }
