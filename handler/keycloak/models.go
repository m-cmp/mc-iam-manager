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
