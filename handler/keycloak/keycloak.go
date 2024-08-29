package keycloak

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/handler"
)

var (
	kc Keycloak
)

func init() {
	kc = Keycloak{
		KcClient:     gocloak.NewClient(handler.KEYCLOAK_HOST),
		Host:         handler.KEYCLOAK_HOST,
		Realm:        handler.KEYCLAOK_REALM,
		Client:       handler.KEYCLAOK_CLIENT,
		ClientSecret: handler.KEYCLAOK_CLIENT_SECRET,
	}
}

func KeycloakGetCerts() (*gocloak.CertResponse, error) {
	ctx := context.Background()
	cert, err := kc.KcClient.GetCerts(ctx, kc.Realm)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return cert, nil
}

// realm-management manage-clients role 필요
func KeycloakGetClientInfo(accessToken string) (*gocloak.Client, error) {
	ctx := context.Background()
	clinetResp, err := kc.KcClient.GetClientRepresentation(ctx, accessToken, kc.Realm, kc.Client)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return clinetResp, nil
}

// Auth Management

func KeycloakLogin(id string, password string) (*gocloak.JWT, error) {
	ctx := context.Background()
	accessTokenResponse, err := kc.KcClient.Login(ctx, kc.Client, kc.ClientSecret, kc.Realm, id, password)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return accessTokenResponse, nil
}

func KeycloakRefreshToken(refreshToken string) (*gocloak.JWT, error) {
	ctx := context.Background()
	accessTokenResponse, err := kc.KcClient.RefreshToken(ctx, refreshToken, kc.Client, kc.ClientSecret, kc.Realm)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return accessTokenResponse, nil
}

func KeycloakLogout(refreshToken string) error {
	ctx := context.Background()
	err := kc.KcClient.Logout(ctx, kc.Client, kc.ClientSecret, kc.Realm, refreshToken)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func KeycloakGetUserInfo(accessToken string) (*gocloak.UserInfo, error) {
	ctx := context.Background()
	userinfo, err := kc.KcClient.GetUserInfo(ctx, accessToken, kc.Realm)
	if err != nil {
		log.Println(err)
	}
	return userinfo, nil
}

func KeycloakTokenInfo(accessToken string) (*gocloak.IntroSpectTokenResult, error) {
	ctx := context.Background()
	userinfo, err := kc.KcClient.RetrospectToken(ctx, accessToken, kc.Client, kc.ClientSecret, kc.Realm)
	if err != nil {
		log.Println(err)
	}
	return userinfo, nil
}

// Users Management
type CreateUserRequset struct {
	Name      string `json:"id"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

func KeycloakCreateUser(accessToken string, user CreateUserRequset, password string) error {
	ctx := context.Background()

	userInfo := gocloak.User{
		Username:  &user.Name,
		Enabled:   gocloak.BoolP(false),
		FirstName: &user.FirstName,
		LastName:  &user.LastName,
		Email:     &user.Email,
	}

	userUUID, err := kc.KcClient.CreateUser(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return err
	}

	err = kc.KcClient.SetPassword(ctx, accessToken, userUUID, kc.Realm, password, false)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

type UserEnableStatusRequest struct {
	UserId string `json:"userid"`
}

func KeycloakActiveUser(accessToken string, userId string) error {
	ctx := context.Background()
	userInfo := gocloak.GetUsersParams{
		Username: &userId,
		Exact:    gocloak.BoolP(true),
	}
	users, err := kc.KcClient.GetUsers(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("user Not Found")
	}

	users[0].Enabled = gocloak.BoolP(true)

	err = kc.KcClient.UpdateUser(ctx, accessToken, kc.Realm, *users[0])
	if err != nil {
		log.Println(err)
		return err
	}

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	res, err := kc.KcClient.GetClientRole(ctx, accessToken, kc.Realm, *clinetResp.ID, "uma_protection")
	if err != nil {
		log.Println(err)
		return err
	}

	roles := []gocloak.Role{*res}
	err = kc.KcClient.AddClientRolesToUser(ctx, accessToken, kc.Realm, *clinetResp.ID, *users[0].ID, roles)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakDeactiveUser(accessToken string, UserId string) error {
	ctx := context.Background()

	userInfo := gocloak.GetUsersParams{
		Username: &UserId,
		Exact:    gocloak.BoolP(true),
	}
	users, err := kc.KcClient.GetUsers(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("user Not Found")
	}

	users[0].Enabled = gocloak.BoolP(false)

	err = kc.KcClient.UpdateUser(ctx, accessToken, kc.Realm, *users[0])
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakGetUsers(accessToken string, userId string) ([]*gocloak.User, error) {
	ctx := context.Background()

	userInfo := gocloak.GetUsersParams{
		Username: &userId,
	}

	users, err := kc.KcClient.GetUsers(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return users, nil
}

func KeycloakDeleteUser(accessToken string, userId string) error {
	ctx := context.Background()

	err := kc.KcClient.DeleteUser(ctx, accessToken, kc.Realm, userId)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakUpdateUser(accessToken string, user CreateUserRequset, userUUID string) error {
	ctx := context.Background()

	userInfo := gocloak.User{
		ID:        &userUUID,
		FirstName: &user.FirstName,
		LastName:  &user.LastName,
		Email:     &user.Email,
	}

	err := kc.KcClient.UpdateUser(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Resource Management

func KeycloakCreateResources(accessToken string, resources CreateResourceRequestArr) (*[]gocloak.ResourceRepresentation, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	result := []gocloak.ResourceRepresentation{}
	createResourceerrors := []error{}
	for _, resource := range resources {
		resname := resource.Framework + ":res:" + resource.OperationId + ":" + resource.Method + ":" + resource.URI
		URIS := []string{resource.URI}
		resreq := gocloak.ResourceRepresentation{
			Name: &resname,
			URIs: &URIS,
		}
		res, err := kc.KcClient.CreateResource(ctx, accessToken, kc.Realm, *clinetResp.ID, resreq)
		if err != nil {
			log.Println(err)
			createResourceerrors = append(createResourceerrors, err)
			continue
		} else {
			_, err := KeycloakCreatePermission(accessToken, resource.Framework, resource.OperationId, resource.OperationId, []string{resname}, []string{})
			if err != nil {
				log.Println(err)
				createResourceerrors = append(createResourceerrors, err)
				continue
			}
		}
		result = append(result, *res)
	}

	if len(createResourceerrors) != 0 {
		return nil, errors.Join(createResourceerrors...)
	}

	return &result, nil
}

func KeycloakCreateMenuResources(accessToken string, resources CreateMenuResourceRequestArr) (*[]gocloak.ResourceRepresentation, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	result := []gocloak.ResourceRepresentation{}
	createResourceerrors := []error{}
	for _, resource := range resources {
		resName := resource.Framework + ":menu:" + resource.Id + ":" + resource.DisplayName + ":" + resource.ParentMenuId + ":" + resource.Priority + ":" + resource.IsAction
		resreq := gocloak.ResourceRepresentation{
			Name: gocloak.StringP(resName),
		}
		res, err := kc.KcClient.CreateResource(ctx, accessToken, kc.Realm, *clinetResp.ID, resreq)
		if err != nil {
			log.Println(err)
			createResourceerrors = append(createResourceerrors, err)
			continue
		} else {
			_, err := KeycloakCreatePermission(accessToken, resource.Framework, resource.Id, resource.Id, []string{resName}, []string{})
			if err != nil {
				log.Println(err)
				createResourceerrors = append(createResourceerrors, err)
				continue
			}
		}
		result = append(result, *res)
	}

	if len(createResourceerrors) != 0 {
		return nil, errors.Join(createResourceerrors...)
	}

	return &result, nil
}

func KeycloakGetResources(accessToken string, params gocloak.GetResourceParams) ([]*gocloak.ResourceRepresentation, error) {
	ctx := context.Background()
	lastNum := 0
	var resources []*gocloak.ResourceRepresentation
	for {
		params.First = gocloak.IntP(lastNum)
		resourcesFetch, err := kc.KcClient.GetResourcesClient(ctx, accessToken, kc.Realm, params)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		resources = append(resources, resourcesFetch...)
		if len(resourcesFetch) < 100 {
			break
		} else {
			lastNum = lastNum + len(resourcesFetch)
		}
	}
	return resources, nil
}

func KeycloakUpdateResources(accessToken string, resourceid string, resource CreateResourceRequest) error {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clientResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return err
	}
	name := resource.Framework + ":" + resource.OperationId + ":" + resource.Method + ":" + resource.URI
	URIS := []string{resource.URI}
	resreq := gocloak.ResourceRepresentation{
		ID:   &resourceid,
		Name: &name,
		URIs: &URIS,
	}
	err = kc.KcClient.UpdateResource(ctx, accessToken, kc.Realm, *clientResp.ID, resreq)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakDeleteResource(accessToken string, resourceid string) error {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clientResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	err = kc.KcClient.DeleteResource(ctx, accessToken, kc.Realm, *clientResp.ID, resourceid)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Role Management

// realm-management manage-realm role require
func KeycloakCreateRole(accessToken string, name string, desc string) (string, error) {
	ctx := context.Background()

	rolereq := gocloak.Role{
		Name:        &name,
		Description: &desc,
	}

	res, err := kc.KcClient.CreateRealmRole(ctx, accessToken, kc.Realm, rolereq)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return res, nil
}

func KeycloakGetRole(accessToken string, name string) (*gocloak.Role, error) {
	ctx := context.Background()

	res, err := kc.KcClient.GetRealmRole(ctx, accessToken, kc.Realm, name)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return res, nil
}

func keycloakGetUserRoles(accessToken string, userUUID string) ([]*gocloak.Role, error) {
	ctx := context.Background()

	roles, err := kc.KcClient.GetRealmRolesByUserID(ctx, accessToken, kc.Realm, userUUID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return roles, nil
}

func KeycloakUpdateRole(accessToken string, name string, desc string) error {
	ctx := context.Background()

	rolereq := gocloak.Role{
		Name:        &name,
		Description: &desc,
	}

	err := kc.KcClient.UpdateRealmRole(ctx, accessToken, kc.Realm, name, rolereq)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakDeleteRole(accessToken string, name string) error {
	ctx := context.Background()

	err := kc.KcClient.DeleteRealmRole(ctx, accessToken, kc.Realm, name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakMappingUserRole(accessToken string, userId string, roleReq string) error {
	ctx := context.Background()

	userInfo := gocloak.GetUsersParams{
		Exact:    gocloak.BoolP(true),
		Username: &userId,
	}
	user, err := kc.KcClient.GetUsers(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return err
	}
	if len(user) != 0 && *user[0].Username != userId {
		err := fmt.Errorf("%s user not found ", userId)
		log.Println(err)
		return err
	}

	role, err := KeycloakGetRole(accessToken, roleReq)
	if err != nil {
		log.Println(err)
		return err
	}

	inputRoles := []gocloak.Role{*role}
	err = kc.KcClient.AddRealmRoleToUser(ctx, accessToken, kc.Realm, *user[0].ID, inputRoles)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakUnMappingUserRole(accessToken string, userId string, roleReq string) error {
	ctx := context.Background()

	userInfo := gocloak.GetUsersParams{
		Exact:    gocloak.BoolP(true),
		Username: &userId,
	}
	user, err := kc.KcClient.GetUsers(ctx, accessToken, kc.Realm, userInfo)
	if err != nil {
		log.Println(err)
		return err
	}
	if len(user) != 0 && *user[0].Username != userId {
		err := fmt.Errorf("%s user not found ", userId)
		log.Println(err)
		return err
	}

	role, err := KeycloakGetRole(accessToken, roleReq)
	if err != nil {
		log.Println(err)
		return err
	}

	inputRoles := []gocloak.Role{*role}
	err = kc.KcClient.DeleteRealmRoleFromUser(ctx, accessToken, kc.Realm, *user[0].ID, inputRoles)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Policy Management

func KeycloakCreatePolicy(accessToken string, name string, desc string) (*gocloak.PolicyRepresentation, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	role, err := KeycloakGetRole(accessToken, name)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	roles := []gocloak.RoleDefinition{{
		ID: role.ID,
	}}

	policyName := name + "Policy"
	policyDesc := desc + " Policy"
	policyType := "role"

	policyreq := gocloak.PolicyRepresentation{
		Name:        &policyName,
		Description: &policyDesc,
		Type:        &policyType,
		RolePolicyRepresentation: gocloak.RolePolicyRepresentation{
			Roles: &roles,
		},
	}

	res, err := kc.KcClient.CreatePolicy(ctx, accessToken, kc.Realm, *clinetResp.ID, policyreq)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return res, nil
}

func KeycloakDeletePolicy(accessToken string, policyId string) error {
	ctx := context.Background()

	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	err = kc.KcClient.DeletePolicy(ctx, accessToken, kc.Realm, *clinetResp.ID, policyId)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakGetPolicies(accessToken string) ([]*gocloak.PolicyRepresentation, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	policyreq := gocloak.GetPolicyParams{
		Type: gocloak.StringP("role"),
	}
	res, err := kc.KcClient.GetPolicies(ctx, accessToken, kc.Realm, *clinetResp.ID, policyreq)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return res, nil
}

// Permission Management

type permissionDetail struct {
	Permission *gocloak.PermissionRepresentation `json:"permission"`
	Resources  []*gocloak.PermissionResource     `json:"resources"`
	Policies   []*gocloak.PolicyRepresentation   `json:"rolePolicies"`
}

func KeycloakCreatePermission(accessToken string, framework string, targetName string, desc string, permissionResources []string, permissionPolicies []string) (*gocloak.PermissionRepresentation, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	permissionName := framework + ":" + targetName
	permissionDesc := desc + " Permission"
	permissionType := "resource"
	permissionReq := gocloak.PermissionRepresentation{
		Name:             &permissionName,
		Description:      &permissionDesc,
		Type:             &permissionType,
		Resources:        &permissionResources,
		Policies:         &permissionPolicies,
		DecisionStrategy: gocloak.AFFIRMATIVE,
	}

	res, err := kc.KcClient.CreatePermission(ctx, accessToken, kc.Realm, *clinetResp.ID, permissionReq)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return res, nil
}

func KeycloakGetPermissions(accessToken string, reqParam gocloak.GetPermissionParams) ([]*gocloak.PermissionRepresentation, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	lastNum := 0
	reqParam.First = gocloak.IntP(lastNum)

	var permissions []*gocloak.PermissionRepresentation
	for {
		reqParam.First = gocloak.IntP(lastNum)
		permission, err := kc.KcClient.GetPermissions(ctx, accessToken, kc.Realm, *clinetResp.ID, reqParam)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		permissions = append(permissions, permission...)
		if len(permission) < 100 {
			break
		}
		lastNum = lastNum + len(permission)
	}

	return permissions, nil
}

func KeycloakGetPermissionDetailByName(accessToken string, framework string, operationid string) (*permissionDetail, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	params := gocloak.GetPermissionParams{
		Name: gocloak.StringP(framework + ":" + operationid),
	}
	permissionRes, err := KeycloakGetPermissions(accessToken, params)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if (len(permissionRes) == 0) || (*permissionRes[0].Name != (framework + ":" + operationid)) {
		return nil, fmt.Errorf("permission not Found")
	}

	resourcesRes, err := kc.KcClient.GetPermissionResources(ctx, accessToken, kc.Realm, *clinetResp.ID, *permissionRes[0].ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	policyRes, err := kc.KcClient.GetAuthorizationPolicyAssociatedPolicies(ctx, accessToken, kc.Realm, *clinetResp.ID, *permissionRes[0].ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	result := &permissionDetail{}
	result.Permission = permissionRes[0]
	result.Resources = resourcesRes
	result.Policies = policyRes

	return result, nil
}

func KeycloakGetPermissionDetail(accessToken string, id string) (*permissionDetail, error) {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	permissionRes, err := kc.KcClient.GetPermission(ctx, accessToken, kc.Realm, *clinetResp.ID, id)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	resourcesRes, err := kc.KcClient.GetPermissionResources(ctx, accessToken, kc.Realm, *clinetResp.ID, id)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	policyRes, err := kc.KcClient.GetAuthorizationPolicyAssociatedPolicies(ctx, accessToken, kc.Realm, *clinetResp.ID, *permissionRes.ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	result := &permissionDetail{}
	result.Permission = permissionRes
	result.Resources = resourcesRes
	result.Policies = policyRes

	return result, nil
}

func KeycloakUpdatePermission(accessToken string, id string, name string, desc string, permissionPolicies []string) error {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	permissionReq := gocloak.PermissionRepresentation{
		ID:          &id,
		Name:        &name,
		Description: &desc,
		Type:        gocloak.StringP("resource"),
		// Resources:        &permissionResources,
		Policies:         &permissionPolicies,
		DecisionStrategy: gocloak.AFFIRMATIVE,
	}

	err = kc.KcClient.UpdatePermission(ctx, accessToken, kc.Realm, *clinetResp.ID, permissionReq)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func KeycloakDeletePermission(accessToken string, id string) error {
	ctx := context.Background()

	//realm-management manage-clients role 필요
	clinetResp, err := KeycloakGetClientInfo(accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	err = kc.KcClient.DeletePermission(ctx, accessToken, kc.Realm, *clinetResp.ID, id)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Ticket Management

func KeycloakGetTicketByRequestUri(accessToken string, uri []string) (*gocloak.JWT, error) {
	ctx := context.Background()

	opt := gocloak.RequestingPartyTokenOptions{
		GrantType:                     gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
		Audience:                      gocloak.StringP(kc.Client),
		Permissions:                   &uri,
		PermissionResourceFormat:      gocloak.StringP("uri"),
		PermissionResourceMatchingURI: gocloak.BoolP(true),
	}

	ticket, err := kc.KcClient.GetRequestingPartyToken(ctx, accessToken, kc.Realm, opt)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return ticket, nil
}

func KeycloakGetAvaliablePermissions(accessToken string) (*[]gocloak.RequestingPartyPermission, error) {
	ctx := context.Background()

	opt := gocloak.RequestingPartyTokenOptions{
		GrantType:    gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
		Audience:     gocloak.StringP(kc.Client),
		ResponseMode: gocloak.StringP("permissions"),
	}

	ticket, err := kc.KcClient.GetRequestingPartyPermissions(ctx, accessToken, kc.Realm, opt)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return ticket, nil
}

type RequestTicket struct {
	Framework   string `json:"framework"`
	OperationId string `json:"operationid"`
	Uri         string `json:"uri"`
}

// https://github.com/keycloak/keycloak/issues/28772
// URI pattern 이 제대로 작동하지 않는 문제가 있음.
func KeycloakGetPermissionTicket(accessToken string, req RequestTicket) (*gocloak.JWT, error) {
	ctx := context.Background()
	params := gocloak.GetResourceParams{}
	if req.Framework != "" && req.OperationId != "" {
		params.Name = gocloak.StringP(req.Framework + ":res:" + req.OperationId)
	}
	if req.Uri != "" {
		params.URI = gocloak.StringP(req.Uri)
	}

	if params.Name != nil {
		resources, err := KeycloakGetResources(accessToken, params)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if len(resources) == 0 {
			return nil, fmt.Errorf("resource Not Found")
		}
		nameArr := []string{}
		for _, resource := range resources {
			nameArr = append(nameArr, *resource.Name)
		}
		opt := gocloak.RequestingPartyTokenOptions{
			GrantType:   gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
			Audience:    gocloak.StringP(kc.Client),
			Permissions: &nameArr,
		}
		ticket, err := kc.KcClient.GetRequestingPartyToken(ctx, accessToken, kc.Realm, opt)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return ticket, nil
	} else if params.URI != nil {
		permissions, err := KeycloakGetAvaliablePermissions(accessToken)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		var targetRes []string
		for _, permission := range *permissions {
			permissionParts := strings.Split(*permission.ResourceName, ":")
			if len(permissionParts) < 4 {
				continue
			}
			if isEqualUri(permissionParts[4], req.Uri) {
				targetRes = append(targetRes, *permission.ResourceName)
				break
			}
		}
		if len(targetRes) == 0 {
			return nil, fmt.Errorf("resource Not Found")
		}
		opt := gocloak.RequestingPartyTokenOptions{
			GrantType:   gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
			Audience:    gocloak.StringP(kc.Client),
			Permissions: &targetRes,
		}
		ticket, err := kc.KcClient.GetRequestingPartyToken(ctx, accessToken, kc.Realm, opt)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return ticket, nil
	}

	return nil, fmt.Errorf("permission not found")
}

func isEqualUri(pattern string, str string) bool {
	regexPattern := regexp.MustCompile(`\{[^/]+\}`).ReplaceAllString(pattern, `[^/]+`)
	regex := regexp.MustCompile("^" + regexPattern + "$")
	return regex.MatchString(str)
}

func KeycloakGetAvailableMenus(accessToken string, framework string) (*[]gocloak.RequestingPartyPermission, error) {
	ctx := context.Background()

	params := gocloak.GetResourceParams{
		Name: gocloak.StringP(framework + ":menu:"),
	}
	resources, err := KeycloakGetResources(accessToken, params)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("menu Not Found")
	}

	names := make([]string, len(resources))
	for i, resource := range resources {
		names[i] = *resource.Name
	}

	opt := gocloak.RequestingPartyTokenOptions{
		GrantType:    gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
		Audience:     gocloak.StringP(kc.Client),
		ResponseMode: gocloak.StringP("permissions"),
		Permissions:  &names,
	}
	ticket, err := kc.KcClient.GetRequestingPartyPermissions(ctx, accessToken, kc.Realm, opt)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return ticket, nil
}
