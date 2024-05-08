package handler

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"mc_iam_manager/iammodels"
)

func CreateUserGroup(ctx buffalo.Context, userGroupReq *iammodels.UserGroupReq) (string, error) {
	createGroup := gocloak.Group{
		Name: &userGroupReq.Name,
		Path: &userGroupReq.Path, //*string              `json:"path,omitempty"`
		//todo subgroup을 어떤식으로 데이터를 받고, 넣어주지?
		// 1. subgroup id를 받아와서 그룹을 생성
		// 2. subgroup name을 받아와서, keycloak으로 부터 조회 후 삽입
		//SubGroups: &userGroupReq.SubGroups,   //*[]Group             `json:"subGroups,omitempty"`
		Attributes:  &userGroupReq.Attributes,  //*map[string][]string `json:"attributes,omitempty"`
		Access:      &userGroupReq.Access,      //*map[string]bool     `json:"access,omitempty"`
		ClientRoles: &userGroupReq.ClientRoles, //*map[string][]string `json:"clientRoles,omitempty"`
		RealmRoles:  &userGroupReq.RealmRoles,  //*[]string            `json:"realmRoles,omitempty"`
	}

	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return "", tokenErr
	}

	if userGroupReq.ParentGroupId != "" {
		return KCClient.CreateChildGroup(ctx, token.AccessToken, KCRealm, userGroupReq.ParentGroupId, createGroup)
	} else {
		return KCClient.CreateGroup(ctx, token.AccessToken, KCRealm, createGroup)
	}
}

func GetUserGroupList(ctx buffalo.Context) ([]*gocloak.Group, error) {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}
	return KCClient.GetGroups(ctx, token.AccessToken, KCRealm, gocloak.GetGroupsParams{})
}

func GetUserGroup(ctx buffalo.Context, groupId string) (*gocloak.Group, error) {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}
	return KCClient.GetGroup(ctx, token.AccessToken, KCRealm, groupId)
}

func DeleteUserGroup(ctx buffalo.Context, groupId string) error {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return tokenErr
	}
	return KCClient.DeleteGroup(ctx, token.AccessToken, KCRealm, groupId)
}

func UpdateUserGroup(ctx buffalo.Context, userGroupInfo iammodels.UserGroupInfo) (*gocloak.Group, error) {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}

	group, err := KCClient.GetGroup(ctx, token.AccessToken, KCRealm, userGroupInfo.GroupId)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	updateGroup := iammodels.UpdateUserGroupByInfoToGroup(userGroupInfo, *group)

	if kcErr := KCClient.UpdateGroup(ctx, token.AccessToken, KCRealm, updateGroup); kcErr != nil {
		return nil, kcErr
	}

	return GetUserGroup(ctx, *updateGroup.ID)
}
