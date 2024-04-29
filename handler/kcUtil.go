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

func GetKeycloakAdminToken(c buffalo.Context) *gocloak.JWT {
	token, kcLoginErr := KCClient.LoginAdmin(c, KCAdmin, KCPwd, KCAdminRealm)

	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}

	cblogger.Info("Tokens : " + token.AccessToken)

	return token
}
