package handler

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"mc_iam_manager/iammodels"
)

func CreateRole(ctx buffalo.Context, bindModel *iammodels.RoleReq) (string, error) {

	keyCloakRole := gocloak.Role{
		Name: &bindModel.RoleName,
	}
	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		return "", tokenErr
	}
	return KCClient.CreateRealmRole(ctx, token.AccessToken, KCRealm, keyCloakRole)
}

func GetRoles(ctx buffalo.Context, searchString string) ([]*gocloak.Role, error) {
	roleSearchParam := gocloak.GetRoleParams{}

	if len(searchString) > 0 {
		roleSearchParam.Search = &searchString
	}

	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)

		return nil, tokenErr
	}

	roleList, err := KCClient.GetRealmRoles(ctx, token.AccessToken, KCRealm, roleSearchParam)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return roleList, err

}

func GetRole(ctx buffalo.Context, roleId string) (*gocloak.Role, error) {

	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}
	return KCClient.GetRealmRoleByID(ctx, token.AccessToken, KCRealm, roleId)
}

func UpdateRole(ctx buffalo.Context, model iammodels.RoleReq) (*gocloak.Role, error) {
	role := iammodels.ConvertRoleReqToRole(model)
	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}
	err := KCClient.UpdateRealmRoleByID(ctx, token.AccessToken, KCRealm, model.ID, role)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return KCClient.GetRealmRoleByID(ctx, token.AccessToken, KCRealm, model.ID)
}

func DeleteRole(ctx buffalo.Context, roleId string) error {

	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return tokenErr
	}

	role, roleErr := KCClient.GetRealmRoleByID(ctx, token.AccessToken, KCRealm, roleId)

	if roleErr != nil {
		cblogger.Info(roleErr)
		return roleErr
	}

	deleteErr := KCClient.DeleteRealmRole(ctx, token.AccessToken, KCRealm, *role.Name)

	if deleteErr != nil {
		cblogger.Info(deleteErr)
		return deleteErr
	}

	return nil

}
