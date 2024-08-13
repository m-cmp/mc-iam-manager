package keycloak

import "github.com/Nerzal/gocloak/v13"

type Keycloak struct {
	KcClient     *gocloak.GoCloak
	Host         string
	Realm        string
	Client       string
	ClientSecret string
}

type UserLogin struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

type UserLoginRefresh struct {
	RefreshToken string `json:"refresh_token"`
}

type UserLogout struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateResourceRequest struct {
	OperationId string `json:"operationId"`
	Method      string `json:"method"`
	Framework   string `json:"framework"`
	URI         string `json:"uri"`
}
type CreateResourceRequestArr []CreateResourceRequest

// https://keycloak.csesmzc.com/admin/realms/mciam/clients/e0630da2-f7ac-4486-a562-7c11bd075ef5/authz/resource-server/policy/be84304d-4392-4748-acce-c4d5296a1df1/associatedPolicies
