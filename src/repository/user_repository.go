package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"

	"github.com/Nerzal/gocloak/v13"
)

type UserRepository struct {
	db             *sql.DB
	keycloakConfig *config.KeycloakConfig
	keycloakClient *gocloak.GoCloak
}

func NewUserRepository(db *sql.DB, keycloakConfig *config.KeycloakConfig, keycloakClient *gocloak.GoCloak) *UserRepository {
	return &UserRepository{
		db:             db,
		keycloakConfig: keycloakConfig,
		keycloakClient: keycloakClient,
	}
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]model.User, error) {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	users, err := r.keycloakClient.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, gocloak.GetUsersParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %v", err)
	}

	var result []model.User
	for _, u := range users {
		user := model.User{
			ID:        *u.ID,
			Username:  *u.Username,
			Email:     *u.Email,
			FirstName: *u.FirstName,
			LastName:  *u.LastName,
		}
		result = append(result, user)
	}

	return result, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	token, err := r.keycloakConfig.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	user, err := r.keycloakClient.GetUserByID(ctx, token.AccessToken, r.keycloakConfig.Realm, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return &model.User{
		ID:        *user.ID,
		Username:  *user.Username,
		Email:     *user.Email,
		FirstName: *user.FirstName,
		LastName:  *user.LastName,
	}, nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	token, err := r.keycloakConfig.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	users, err := r.keycloakClient.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, gocloak.GetUsersParams{
		Username: gocloak.StringP(username),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	user := users[0]
	return &model.User{
		ID:        *user.ID,
		Username:  *user.Username,
		Email:     *user.Email,
		FirstName: *user.FirstName,
		LastName:  *user.LastName,
	}, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	keycloakUser := gocloak.User{
		Username:      &user.Username,
		Email:         &user.Email,
		FirstName:     &user.FirstName,
		LastName:      &user.LastName,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	userID, err := r.keycloakClient.CreateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, keycloakUser)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	user.ID = userID
	return nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	keycloakUser := gocloak.User{
		ID:            &user.ID,
		Username:      &user.Username,
		Email:         &user.Email,
		FirstName:     &user.FirstName,
		LastName:      &user.LastName,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	err = r.keycloakClient.UpdateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, keycloakUser)
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	return nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	err = r.keycloakClient.DeleteUser(ctx, token.AccessToken, r.keycloakConfig.Realm, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}
