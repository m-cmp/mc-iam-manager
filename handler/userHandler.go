package handler

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"mc_iam_manager/iammodels"
)

func CreateUser(ctx buffalo.Context, userReq *iammodels.UserReq) (string, error) {

	user := gocloak.User{
		FirstName: &userReq.UserFirstName,
		LastName:  &userReq.UserLastName,
		Email:     &userReq.Email,
		Enabled:   gocloak.BoolP(true),
		Username:  &userReq.UserName,
	}

	return KCClient.CreateUser(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, user)
}

func GetUserList(ctx buffalo.Context) ([]*gocloak.User, error) {
	return KCClient.GetUsers(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, gocloak.GetUsersParams{})
}

func GetUser(ctx buffalo.Context, userId string) (*gocloak.User, error) {
	return KCClient.GetUserByID(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, userId)
}

func DeleteUser(ctx buffalo.Context, userId string) error {
	return KCClient.DeleteUser(ctx, GetKeycloakAdminToken(ctx).AccessToken, KCRealm, userId)
}

func UpdateUser(ctx buffalo.Context, userInfo iammodels.UserInfo) error {
	adminAccessToken := GetKeycloakAdminToken(ctx).AccessToken
	user, err := KCClient.GetUserByID(ctx, adminAccessToken, KCRealm, userInfo.UserId)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	/**
	To-do
	User Update 항목 logic 추가
	*/
	user.Username = &userInfo.UserName

	return KCClient.UpdateUser(ctx, adminAccessToken, KCRealm, *user)
}
