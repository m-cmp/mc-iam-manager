package handler

import (
	"context"
	"github.com/Nerzal/gocloak/v13"
	"mc_iam_manager/iammodels"
	"net/http"
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

func CreateRole(bindModel *iammodels.RoleReq) map[string]interface{} {
	ctx := context.Background() // 기본 컨텍스트 생성
	cblogger.Info("CreateRole")
	cblogger.Info(*bindModel)

	token, kcLoginErr := KCClient.LoginAdmin(ctx, KCAdmin, KCPwd, KCAdminRealm)

	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}

	cblogger.Info("Tokens : " + token.AccessToken)

	keyCloakRole := gocloak.Role{
		Name: &bindModel.RoleName,
	}

	_, kcErr := KCClient.CreateRealmRole(ctx, token.AccessToken, KCRealm, keyCloakRole)

	if kcErr != nil {
		return map[string]interface{}{
			"message": kcErr,
			"status":  http.StatusBadRequest,
		}
	}

	//start of client Role

	//param := gocloak.GetClientsParams{}
	//kcClientList, _ := KCClient.GetClients(ctx, token.AccessToken, KCRealm, param)
	//
	//for _, client := range kcClientList {
	//	if KCClientID == *client.ClientID {
	//		_, kcErr := KCClient.CreateClientRole(ctx, token.AccessToken, KCRealm, *client.ID, keyCloakRole)
	//		if kcErr != nil {
	//			return map[string]interface{}{
	//				"message": kcErr,
	//				"status":  http.StatusBadRequest,
	//			}
	//		}
	//		break
	//	}
	//}

	//end of client Role

	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}
}

func GetRoles(searchString string) []*gocloak.Role {

	ctx := context.Background() // 기본 컨텍스트 생성
	cblogger.Info("CreateRole")

	token, kcLoginErr := KCClient.LoginAdmin(ctx, KCAdmin, KCPwd, KCAdminRealm)

	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}

	roleSearchParam := gocloak.GetRoleParams{}

	if len(searchString) > 0 {
		roleSearchParam.Search = &searchString
	}

	roleList, _ := KCClient.GetRealmRoles(ctx, token.AccessToken, KCRealm, roleSearchParam)

	cblogger.Info(roleList)

	return roleList
}

func GetRole(roleId string) *gocloak.Role {
	cblogger.Info("roleId : " + roleId)
	c := context.Background()
	token, kcLoginErr := KCClient.LoginAdmin(c, KCAdmin, KCPwd, KCAdminRealm)

	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}
	//role, err := KCClient.GetRealmRole(c, token.AccessToken, KCRealm, roleId)
	role, err := KCClient.GetRealmRoleByID(c, token.AccessToken, KCRealm, roleId)
	cblogger.Info(role)

	if err != nil {
		cblogger.Error(err)
	}

	return role
}

func UpdateRole(model iammodels.RoleReq) error {
	cblogger.Info(model)
	//cblogger.Info("roleId : " + roleId)
	c := context.Background()
	token, kcLoginErr := KCClient.LoginAdmin(c, KCAdmin, KCPwd, KCAdminRealm)

	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}

	role := iammodels.ConvertRoleReqToRole(model)
	cblogger.Info(model.ID)
	cblogger.Info(&role)
	err := KCClient.UpdateRealmRoleByID(c, token.AccessToken, KCRealm, model.ID, role)

	if err != nil {
		cblogger.Error(err)
	}

	return err
}

func DeleteRole(roleId string) error {
	ctx := context.Background() // 기본 컨텍스트 생성

	token, kcLoginErr := KCClient.LoginAdmin(ctx, KCAdmin, KCPwd, KCAdminRealm)

	if kcLoginErr != nil {
		cblogger.Info(kcLoginErr)
	}

	role, roleErr := KCClient.GetRealmRoleByID(ctx, token.AccessToken, KCRealm, roleId)

	if roleErr != nil {
		cblogger.Info(roleErr)
	}

	return KCClient.DeleteRealmRole(ctx, token.AccessToken, KCRealm, *role.Name)

}
