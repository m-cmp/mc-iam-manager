package actions

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/m-cmp/mc-iam-manager/handler"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
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

func SyncRoleListWithKeycloak(c buffalo.Context) error {
	err := deletAllUnlinkedRoles(c)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = updateRolesFromeKeycloak(c)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, nil)

}

func deletAllUnlinkedRoles(c buffalo.Context) error {

	accessToken := c.Value("accessToken").(string)

	// kcRoles, err := keycloak.KeycloakGetRoles(accessToken)
	// if err != nil {
	// 	log.Println(err)
	// 	return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	// }

	tx := c.Value("tx").(*pop.Connection)
	roles, err := handler.GetRoleList(tx)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return err
	}
	errs := make(map[string][]error)
	for _, role := range *roles {
		var errArr []error
		policy, err := keycloak.KeycloakGetPolicy(accessToken, role.Policy)
		if err != nil {
			log.Println(err)
			errArr = append(errArr, err)
		}
		if policy == nil {
			err = handler.DeleteWorkspaceUserRoleMappingByRoleId(tx, role.ID.String())
			if err != nil {
				log.Println(err)
				errArr = append(errArr, err)
			}
			err = handler.DeleteRole(tx, &role)
			if err != nil {
				log.Println(err)
				errArr = append(errArr, err)
			}
		} else {

		}
		errs[role.Name] = errArr
	}

	// err = handler.DeleteRole(tx, s)
	// if err != nil {
	// 	log.Println(err)
	// 	err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
	// 	return err

	// }

	// err = keycloak.KeycloakDeletePolicy(accessToken, s.Policy)
	// if err != nil {
	// 	log.Println(err)
	// 	return err

	// }

	// err = keycloak.KeycloakDeleteRole(accessToken, s.Name)
	// if err != nil {
	// 	log.Println(err)
	// 	return err

	// }

	return nil
}

func updateRolesFromeKeycloak(c buffalo.Context) error {
	token := c.Value("accessToken").(string)

	roles, err := keycloak.KeycloakGetRoles(token)
	if err != nil {
		log.Println(err)
		return err
	}

	policies, err := keycloak.KeycloakGetPolicies(token)
	if err != nil {
		log.Println(err)
		return err
	}

	tx := c.Value("tx").(*pop.Connection)
	for _, role := range roles {
		var s models.Role
		s.Name = *role.Name
		s.Description = nulls.String{
			String: *role.Description,
			Valid:  true,
		}
		for _, policy := range policies {
			for _, policyRole := range *policy.Roles {
				if *policyRole.ID == *role.ID {
					s.Policy = *policy.ID
					break
				}
			}
		}
		if s.Policy == "" {
			break
		}
		_, err := handler.CreateRole(tx, &s)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
