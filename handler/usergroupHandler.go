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

	if userGroupReq.ParentGroupId != "" {
		return KCClient.CreateChildGroup(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, userGroupReq.ParentGroupId, createGroup)
	} else {
		return KCClient.CreateGroup(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, createGroup)
	}
}

func GetUserGroupList(ctx buffalo.Context) ([]*gocloak.Group, error) {
	return KCClient.GetGroups(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, gocloak.GetGroupsParams{})
}

func GetUserGroup(ctx buffalo.Context, groupId string) (*gocloak.Group, error) {
	return KCClient.GetGroup(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, groupId)
}

func DeleteUserGroup(ctx buffalo.Context, groupId string) error {
	return KCClient.DeleteGroup(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, groupId)
}

func UpdateUserGroup(ctx buffalo.Context, userGroupInfo iammodels.UserGroupInfo) error {
	adminAccessToken := GetKeycloakAdminToken(ctx).AccessToken
	group, err := KCClient.GetGroup(ctx, adminAccessToken, KCRealm, userGroupInfo.GroupId)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	updateGroup := iammodels.UpdateUserGroupByInfoToGroup(userGroupInfo, *group)

	return KCClient.UpdateGroup(ctx, adminAccessToken, KCRealm, updateGroup)
}
