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
	if nsArray, ok := nsList["ns"].([]interface{}); ok {
		for _, item := range nsArray {
			if nsItem, ok := item.(map[string]interface{}); ok {
				var s models.Project
				s.NsID = nsItem["id"].(string)
				s.Name = nsItem["name"].(string)
				if description, ok := nsItem["description"].(nulls.String); ok {
					s.Description = description
				}
				_, err := handler.CreateProject(tx, &s)
				if err != nil {
					log.Println(err)
					return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
				}
			}
		}
	}
	projectList, err := handler.GetProjectList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(projectList))
}
