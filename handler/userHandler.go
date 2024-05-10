package handler

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/buffalo"
	"mc_iam_manager/iammodels"
)

func CreateUser(ctx buffalo.Context, userReq *iammodels.UserReq) (string, error) {

	token, tokenErr := GetKeycloakAdminToken(ctx)
	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return "", tokenErr
	}

	user := gocloak.User{
		FirstName: &userReq.UserFirstName,
		LastName:  &userReq.UserLastName,
		Email:     &userReq.Email,
		Enabled:   gocloak.BoolP(true),
		Username:  &userReq.UserName,
	}

	return KCClient.CreateUser(ctx, token.AccessToken, KCRealm, user)
}

func GetUserList(ctx buffalo.Context) ([]*gocloak.User, error) {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}

	return KCClient.GetUsers(ctx, token.AccessToken, KCRealm, gocloak.GetUsersParams{})
}

func GetUser(ctx buffalo.Context, userId string) (*gocloak.User, error) {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}
	return KCClient.GetUserByID(ctx, token.AccessToken, KCRealm, userId)
}

func DeleteUser(ctx buffalo.Context, userId string) error {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return tokenErr
	}
	return KCClient.DeleteUser(ctx, token.AccessToken, KCRealm, userId)
}

func UpdateUser(ctx buffalo.Context, userInfo iammodels.UserInfo) (*gocloak.User, error) {
	token, tokenErr := GetKeycloakAdminToken(ctx)

	if tokenErr != nil {
		cblogger.Error(tokenErr)
		return nil, tokenErr
	}

	user, err := KCClient.GetUserByID(ctx, token.AccessToken, KCRealm, userInfo.Id)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	/**
	To-do
	User Update 항목 logic 추가
	*/
	user.Email = &userInfo.Email
	updateErr := KCClient.UpdateUser(ctx, token.AccessToken, KCRealm, *user)
	if updateErr != nil {
		cblogger.Error(updateErr)
		return nil, updateErr
	}
	return KCClient.GetUserByID(ctx, token.AccessToken, KCRealm, userInfo.Id)
}
