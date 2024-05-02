package handler

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"mc_iam_manager/iammodels"
	"net/http"
)

func CreateRole(ctx buffalo.Context, bindModel *iammodels.RoleReq) map[string]interface{} {

	keyCloakRole := gocloak.Role{
		Name: &bindModel.RoleName,
	}

	_, kcErr := KCClient.CreateRealmRole(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, keyCloakRole)

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

func GetRoles(ctx buffalo.Context, searchString string) []*gocloak.Role {
	roleSearchParam := gocloak.GetRoleParams{}

	if len(searchString) > 0 {
		roleSearchParam.Search = &searchString
	}

	roleList, _ := KCClient.GetRealmRoles(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, roleSearchParam)

	return roleList
}

func GetRole(ctx buffalo.Context, roleId string) *gocloak.Role {

	role, err := KCClient.GetRealmRoleByID(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, roleId)
	cblogger.Info(role)

	if err != nil {
		cblogger.Error(err)
	}

	return role
}

func UpdateRole(ctx buffalo.Context, model iammodels.RoleReq) error {
	role := iammodels.ConvertRoleReqToRole(model)
	err := KCClient.UpdateRealmRoleByID(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, model.ID, role)

	if err != nil {
		cblogger.Error(err)
	}

	return err
}

func DeleteRole(ctx buffalo.Context, roleId string) error {
	adminAccessToken := GetKeycloakAdminToken(ctx).AccessToken
	role, roleErr := KCClient.GetRealmRoleByID(ctx, adminAccessToken, KCRealm, roleId)

	if roleErr != nil {
		cblogger.Info(roleErr)
	}

	return KCClient.DeleteRealmRole(ctx, adminAccessToken, KCRealm, *role.Name)

}
