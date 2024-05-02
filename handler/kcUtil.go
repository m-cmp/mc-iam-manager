package handler

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"os"
)

var KCAdmin = os.Getenv("keycloakAdmin")
var KCPwd = os.Getenv("keycloakAdminPwd")
var KCUri = os.Getenv("keycloakHost")
var KCClientID = os.Getenv("keycloakClient")
var KCClientSecret = os.Getenv("keycloakClientSecret")
var KCAdminRealm = os.Getenv("keycloakAdminRealm")
var KCRealm = os.Getenv("keycloakRealm")
var KCClient = gocloak.NewClient(KCUri)

var adminToken gocloak.JWT

func GetKeycloakAdminToken(c buffalo.Context) *gocloak.JWT {
	//todo
	// 1. admintoken expire chk
	// 1-1. if expired
	// 2-1. admin token refresh
	// 3-1. return token
	// 1-2. if not expired
	// 2-2. return admin token

	token, kcLoginErr := KCClient.LoginAdmin(c, KCAdmin, KCPwd, KCAdminRealm)
	adminToken = *token
	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}

	//cblogger.Info("Tokens : " + token.AccessToken)

	return &adminToken
}
