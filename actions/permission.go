package actions

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

type createPermissionRequset struct {
	// Framework           string   `json:"framework"`  // deprecated
	// Name                string   `json:"name"`  // deprecated
	Desc string `json:"desc"`
	// PermissionResources []string `json:"resources"` // deprecated
	PermissionPolicies []string `json:"role"`
}

// func CreatePermission(c buffalo.Context) error {
// 	accessToken := c.Value("accessToken").(string)
// 	permissionReq := &createPermissionRequset{}
// 	if err := c.Bind(permissionReq); err != nil {
// 		log.Println(err)
// 		return c.Render(http.StatusBadRequest, r.JSON(err))
// 	}
// 	resoruces, err := keycloak.KeycloakCreatePermission(accessToken, permissionReq.Framework, permissionReq.Name, permissionReq.Desc, permissionReq.PermissionResources, permissionReq.PermissionPolicies)
// 	if err != nil {
// 		log.Println(err)
// 		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
// 	}
// 	return c.Render(http.StatusOK, r.JSON(resoruces))
// }

func ImportPermissionByCsv(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)
	f, err := c.File("file")
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}
	reader := csv.NewReader(f.File)
	records, err := reader.ReadAll()
	if err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	framework := c.Param("framework")
	totalPermissionCount := len(records) - 1
	updatedPermissionCount := 0
	var errs []string
	policycols := records[0][2:]
	for _, record := range records[1:] {
		if framework != "all" && record[0] != framework {
			continue
		}
		permission, err := keycloak.KeycloakGetPermissionDetailByName(accessToken, record[0], record[1])
		if err != nil {
			log.Println(err)
			errs = append(errs, err.Error()+":"+record[0]+":"+record[1])
			continue
		}
		policiesArr := []string{}
		for i, policy := range record[2:] {
			if yn, _ := strconv.ParseBool(policy); yn {
				policiesArr = append(policiesArr, policycols[i])
			}
		}
		err = keycloak.KeycloakUpdatePermission(accessToken, *permission.Permission.ID, *permission.Permission.Name, *permission.Permission.Description, policiesArr)
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			updatedPermissionCount++
		}
	}
	resultMsg := fmt.Sprintf("permission import Success %d / %d", updatedPermissionCount, totalPermissionCount)

	if len(errs) != 0 {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": strings.Join(errs, "; "), "status": resultMsg}))
	}
	return c.Render(http.StatusOK, r.JSON((map[string]string{"status": resultMsg})))
}

func GetCurrentPermissionCsv(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Param("framework")

	policies, err := keycloak.KeycloakGetPolicies(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}
	policiesName := make([]string, len(policies))
	for i, policy := range policies {
		policiesName[i] = *policy.Name
	}

	params := gocloak.GetPermissionParams{}
	if framework != "all" {
		params.Name = gocloak.StringP(framework + ":")
	}

	permissions, err := keycloak.KeycloakGetPermissions(accessToken, params)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	rows := [][]string{{"framework", "resource"}}
	rows[0] = append(rows[0], policiesName...)

	for _, permission := range permissions {
		permissionParts := strings.Split(*permission.Name, ":")
		permissionDetail, err := keycloak.KeycloakGetPermissionDetailByName(accessToken, permissionParts[0], permissionParts[1])
		if err != nil {
			log.Println(err)
			return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
		}
		policiesArr := make([]string, len(policies))
		for _, policy := range permissionDetail.Policies {

			for i, v := range policiesName {
				if *policy.Name == v {
					policiesArr[i] = "true"
				}
			}
		}
		row := []string{permissionParts[0], permissionParts[1]}
		row = append(row, policiesArr...)
		rows = append(rows, row)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	for _, record := range rows {
		if err := w.Write(record); err != nil {
			log.Println(err)
			return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
		}
	}
	w.Flush()

	if err := w.Error(); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.Download(c, "currentPermission.csv", &buf))
}

func GetPermissions(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)
	name := c.Request().URL.Query().Get("name")
	params := gocloak.GetPermissionParams{
		Name: &name,
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

	framework := c.Param("framework")
	operationid := c.Param("operationid")

	permissions, err := keycloak.KeycloakGetPermissionDetailByName(accessToken, framework, operationid)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(permissions))
}

// func UpdatePermission(c buffalo.Context) error {
// 	accessToken := c.Value("accessToken").(string)
// 	permissionId := c.Param("permissionid")
// 	permissionReq := &createPermissionRequset{}
// 	if err := c.Bind(permissionReq); err != nil {
// 		log.Println(err)
// 		return c.Render(http.StatusBadRequest, r.JSON(err))
// 	}
// 	err := keycloak.KeycloakUpdatePermission(accessToken, permissionId, permissionReq.Name, permissionReq.Desc, permissionReq.PermissionResources, permissionReq.PermissionPolicies)
// 	if err != nil {
// 		log.Println(err)
// 		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
// 	}
// 	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "success"}))
// }

func UpdateResourcePermissionByOperationId(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	framework := c.Param("framework")
	operationid := c.Param("operationid")

	params := gocloak.GetPermissionParams{
		Name: gocloak.StringP(framework + ":" + operationid),
	}
	permissions, err := keycloak.KeycloakGetPermissions(accessToken, params)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(err))
	}
	if (len(permissions) == 0) || (*permissions[0].Name != (framework + ":" + operationid)) {
		errmsg := fmt.Errorf("permission not Found")
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": errmsg.Error()}))
	}

	permissionReq := &createPermissionRequset{}
	if err := c.Bind(permissionReq); err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"errors": err.Error()}))
	}

	for i, v := range permissionReq.PermissionPolicies {
		permissionReq.PermissionPolicies[i] = v + "Policy"
	}

	err = keycloak.KeycloakUpdatePermission(accessToken, *permissions[0].ID, framework+":"+operationid, permissionReq.Desc, permissionReq.PermissionPolicies)
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
