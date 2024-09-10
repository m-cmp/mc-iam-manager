package actions

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/m-cmp/mc-iam-manager/handler"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
	"github.com/m-cmp/mc-iam-manager/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type createRoleRequset struct {
	Name         string       `json:"name" db:"name"`
	Description  nulls.String `json:"description" db:"description"`
	PlatformRole string       `json:"platformRole"`
}

var (
	platformRolePrefix = "platform-"
)

func CreateRole(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	var req createRoleRequset
	var s models.Role
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	err = handler.CopyStruct(req, &s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	if yn, _ := strconv.ParseBool(req.PlatformRole); yn {
		s.Name = platformRolePrefix + req.Name
	}

	_, err = keycloak.KeycloakCreateRole(accessToken, s.Name, s.Description.String)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	policy, err := keycloak.KeycloakCreatePolicy(accessToken, s.Name, s.Description.String)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	s.Policy = *policy.ID

	tx := c.Value("tx").(*pop.Connection)
	roleRes, err := handler.CreateRole(tx, &s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "Role is already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(roleRes))
}

func platformRoleParser(resRoles *models.Roles, isPlatformRole bool) models.Roles {
	var resultRoles models.Roles
	prefixCheck := strings.HasPrefix
	if !isPlatformRole {
		prefixCheck = func(s, prefix string) bool { return !strings.HasPrefix(s, prefix) }
	}

	for _, role := range *resRoles {
		if prefixCheck(role.Name, platformRolePrefix) {
			resultRoles = append(resultRoles, role)
		}
	}

	return resultRoles
}

func SearchRolesByName(c buffalo.Context) error {
	var err error
	roleName := c.Param("roleName")
	option := c.Request().URL.Query().Get("option")
	platformRole, _ := strconv.ParseBool(c.Request().URL.Query().Get("platformRole"))

	tx := c.Value("tx").(*pop.Connection)
	resRoles, err := handler.SearchRolesByName(tx, roleName, option)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	resultRoles := platformRoleParser(resRoles, platformRole)

	return c.Render(http.StatusOK, r.JSON(resultRoles))
}

func GetRoleList(c buffalo.Context) error {
	var err error
	tx := c.Value("tx").(*pop.Connection)
	platformRole, _ := strconv.ParseBool(c.Request().URL.Query().Get("platformRole"))

	res, err := handler.GetRoleList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	resultRoles := platformRoleParser(res, platformRole)

	return c.Render(http.StatusOK, r.JSON(resultRoles))
}

func GetRoleById(c buffalo.Context) error {
	var err error
	roleId := c.Param("roleId")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetRoleById(tx, uuid.FromStringOrNil(roleId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetRoleByPolicyId(c buffalo.Context) error {
	var err error
	policyId := c.Param("policyId")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetRoleByPolicyId(tx, policyId)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

type updateRoleByIdRequest struct {
	Description nulls.String `json:"description" db:"description"`
}

func UpdateRoleById(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	var req updateRoleByIdRequest
	var err error

	roleId := c.Param("roleId")
	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetRoleById(tx, uuid.FromStringOrNil(roleId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = handler.CopyStruct(req, s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	res, err := handler.UpdateRole(tx, s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = keycloak.KeycloakUpdateRole(accessToken, res.Name, res.Description.String)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(res))
}

func DeleteRoleById(c buffalo.Context) error {
	accessToken := c.Value("accessToken").(string)

	var err error
	roleId := c.Param("roleId")
	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetRoleById(tx, uuid.FromStringOrNil(roleId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	err = handler.DeleteRole(tx, s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Role is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = keycloak.KeycloakDeletePolicy(accessToken, s.Policy)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = keycloak.KeycloakDeleteRole(accessToken, s.Name)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "ID(" + s.ID.String() + ") / Name(" + s.Name + ") is delected.."}))
}
