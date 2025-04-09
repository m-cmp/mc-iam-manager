package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

type UserService struct {
	userRepo          *repository.UserRepository
	platformRoleRepo  *repository.PlatformRoleRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	tokenRepo         *repository.TokenRepository
	keycloakClient    *gocloak.GoCloak
	keycloakConfig    *config.KeycloakConfig
}

func NewUserService(userRepo *repository.UserRepository, keycloakConfig *config.KeycloakConfig, keycloakClient *gocloak.GoCloak) *UserService {
	return &UserService{
		userRepo:       userRepo,
		keycloakClient: keycloakClient,
		keycloakConfig: keycloakConfig,
	}
}

func (s *UserService) getValidToken(ctx context.Context) (string, error) {
	// DB에서 유효한 토큰 확인
	token, err := s.tokenRepo.GetTokenByUserID(s.keycloakConfig.ClientID)
	if err == nil {
		// 토큰이 만료 10분 전이면 새로 발급
		if time.Until(token.ExpiresAt) > 10*time.Minute {
			return token.Token, nil
		}
	}

	// 유효한 토큰이 없거나 곧 만료되는 경우 새로 발급
	tokenResponse, err := s.keycloakClient.LoginClient(ctx, s.keycloakConfig.ClientID, s.keycloakConfig.ClientSecret, s.keycloakConfig.Realm)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	// 토큰 저장
	if err := s.tokenRepo.SaveToken(s.keycloakConfig.ClientID, tokenResponse.AccessToken, int64(tokenResponse.ExpiresIn)); err != nil {
		return "", fmt.Errorf("failed to save token: %v", err)
	}

	return tokenResponse.AccessToken, nil
}

// GetUsers returns a list of users
func (s *UserService) GetUsers(ctx context.Context) ([]model.User, error) {
	token, err := s.keycloakConfig.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("관리자 로그인 실패: %v", err)
	}

	users, err := s.keycloakClient.GetUsers(ctx, token.AccessToken, s.keycloakConfig.Realm, gocloak.GetUsersParams{})
	if err != nil {
		return nil, fmt.Errorf("사용자 목록 조회 실패: %v", err)
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

// GetUserByID returns a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

// GetUserByUsername returns a user by username from Keycloak
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(ctx, username)
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *model.User) error {
	return s.userRepo.CreateUser(ctx, user)
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, user *model.User) error {
	return s.userRepo.UpdateUser(ctx, user)
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.DeleteUser(ctx, id)
}

// GetUser returns a user by ID
func (s *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}
