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
	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		return map[string]interface{}{
			"message": tokenErr,
			"status":  http.StatusBadRequest,
		}
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

func GetRoles(ctx buffalo.Context, searchString string) map[string]interface{} {
	roleSearchParam := gocloak.GetRoleParams{}

	if len(searchString) > 0 {
		roleSearchParam.Search = &searchString
	}

	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)

		return map[string]interface{}{
			"error":  tokenErr,
			"status": http.StatusInternalServerError,
		}
	}

	roleList, _ := KCClient.GetRealmRoles(ctx, token.AccessToken, KCRealm, roleSearchParam)

	return map[string]interface{}{
		"roleList": roleList,
		"status":   http.StatusOK,
	}

}

func GetRole(ctx buffalo.Context, roleId string) (*gocloak.Role, error) {

	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}

	role, err := KCClient.GetRealmRoleByID(ctx, token.AccessToken, KCRealm, roleId)
	cblogger.Info(role)

	if err != nil {
		cblogger.Error(err)

		return nil, err
	}

	return role, err
}

func UpdateRole(ctx buffalo.Context, model iammodels.RoleReq) map[string]interface{} {
	role := iammodels.ConvertRoleReqToRole(model)
	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return map[string]interface{}{
			"error":  tokenErr,
			"status": http.StatusInternalServerError,
		}
	}
	err := KCClient.UpdateRealmRoleByID(ctx, token.AccessToken, KCRealm, model.ID, role)

	if err != nil {
		cblogger.Error(err)
		return map[string]interface{}{
			"error":  err,
			"status": http.StatusInternalServerError,
		}
	}

	return map[string]interface{}{
		"message": "update successfully",
		"status":  http.StatusOK,
	}
}

func DeleteRole(ctx buffalo.Context, roleId string) map[string]interface{} {

	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return map[string]interface{}{
			"error":  tokenErr,
			"status": http.StatusInternalServerError,
		}
	}

	role, roleErr := KCClient.GetRealmRoleByID(ctx, token.AccessToken, KCRealm, roleId)

	if roleErr != nil {
		cblogger.Info(roleErr)
		return map[string]interface{}{
			"error":  roleErr,
			"status": http.StatusInternalServerError,
		}
	}

	deleteErr := KCClient.DeleteRealmRole(ctx, token.AccessToken, KCRealm, *role.Name)

	if deleteErr != nil {
		cblogger.Info(deleteErr)
		return map[string]interface{}{
			"error":  deleteErr,
			"status": http.StatusInternalServerError,
		}
	}

	return map[string]interface{}{
		"message": "Delete successfully",
		"status":  http.StatusOK,
	}

}
