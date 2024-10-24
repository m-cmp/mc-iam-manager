package keycloak

import (
	"os"

	"github.com/Nerzal/gocloak/v13"
)

var COMPANY_NAME = os.Getenv("COMPANY_NAME")

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
	Framework   string `json:"framework"`
	OperationId string `json:"operationId"`
	Method      string `json:"method"`
	URI         string `json:"uri"`
}
type CreateResourceRequestArr []CreateResourceRequest
type CreateMenuResourceRequest struct {
	Framework   string `json:"framework"`
	Id          string `json:"id"`
	ParentId    string `json:"parentid"`
	DisplayName string `json:"displaymame"`
	IsAction    string `json:"isaction"`
	ResType     string `json:"restype"`
	Priority    string `json:"priority"`
	MenuNumber  string `json:"menuno"`
}
type CreateMenuResourceRequestArr []CreateMenuResourceRequest
