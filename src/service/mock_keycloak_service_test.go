package service

import (
	"context"

	"github.com/Nerzal/gocloak/v13"
	"github.com/golang-jwt/jwt/v5"
	"github.com/m-cmp/mc-iam-manager/model"
)

// mockKeycloakService 테스트용 KeycloakService 스텁 (모든 메서드 nil/에러 반환)
type mockKeycloakService struct{}

func (m *mockKeycloakService) KeycloakAdminLogin(ctx context.Context) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetUser(ctx context.Context, kcId string) (*gocloak.User, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetUserByUsername(ctx context.Context, username string) (*gocloak.User, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetUsers(ctx context.Context, enabled *bool) ([]*gocloak.User, error) {
	return nil, nil
}
func (m *mockKeycloakService) CreateUser(ctx context.Context, user *model.User) (string, error) {
	return "", nil
}
func (m *mockKeycloakService) UpdateUser(ctx context.Context, user *model.User) error { return nil }
func (m *mockKeycloakService) DeleteUser(ctx context.Context, kcId string) error      { return nil }
func (m *mockKeycloakService) EnableUser(ctx context.Context, kcUserID string) error  { return nil }
func (m *mockKeycloakService) CheckAdminLogin(ctx context.Context) (bool, error)      { return false, nil }
func (m *mockKeycloakService) CheckRealm(ctx context.Context) (bool, error)           { return false, nil }
func (m *mockKeycloakService) CreateRealm(ctx context.Context, accessToken string) (bool, error) {
	return false, nil
}
func (m *mockKeycloakService) ExistRealm(ctx context.Context, accessToken string) (bool, error) {
	return false, nil
}
func (m *mockKeycloakService) ExistClient(ctx context.Context, accessToken string) (bool, error) {
	return false, nil
}
func (m *mockKeycloakService) CheckClient(ctx context.Context) (bool, error) { return false, nil }
func (m *mockKeycloakService) CreateClient(ctx context.Context, accessToken string) (bool, error) {
	return false, nil
}
func (m *mockKeycloakService) GetCerts(ctx context.Context) (*gocloak.CertResponse, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetUserIDFromToken(ctx context.Context, token *gocloak.JWT) (string, error) {
	return "", nil
}
func (m *mockKeycloakService) Login(ctx context.Context, username, password string) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) RefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) EnsureGroupExistsAndAssignUser(ctx context.Context, kcUserId, groupName string) error {
	return nil
}
func (m *mockKeycloakService) RemoveUserFromGroup(ctx context.Context, kcUserId, groupName string) error {
	return nil
}
func (m *mockKeycloakService) GetRequestingPartyToken(ctx context.Context, accessToken string, options gocloak.RequestingPartyTokenOptions) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) ValidateTokenAndGetClaims(ctx context.Context, token string) (*jwt.MapClaims, error) {
	return nil, nil
}
func (m *mockKeycloakService) SetupInitialKeycloakAdmin(ctx context.Context, adminToken *gocloak.JWT) (string, error) {
	return "", nil
}
func (m *mockKeycloakService) CheckUserRoles(ctx context.Context, username string) error { return nil }
func (m *mockKeycloakService) GetUserPermissions(ctx context.Context, roles []string) ([]string, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetImpersonationToken(ctx context.Context) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetImpersonationTokenByAdminToken(ctx context.Context, userID string, targetClientID string) (string, error) {
	return "", nil
}
func (m *mockKeycloakService) GetImpersonationTokenByServiceAccount(ctx context.Context) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) GetSamlAssertionByServiceAccount(ctx context.Context, samlClientAudience string) (string, error) {
	return "", nil
}
func (m *mockKeycloakService) AssignRealmRoleToUser(ctx context.Context, kcUserId, roleName string) error {
	return nil
}
func (m *mockKeycloakService) CheckRealmRoleExists(ctx context.Context, roleName string) (bool, error) {
	return false, nil
}
func (m *mockKeycloakService) CreateRealmRole(ctx context.Context, roleName string) error { return nil }
func (m *mockKeycloakService) CreateRealmRoleAndWait(ctx context.Context, roleName string) error {
	return nil
}
func (m *mockKeycloakService) RemoveRealmRoleFromUser(ctx context.Context, kcUserId, roleName string) error {
	return nil
}
func (m *mockKeycloakService) IsRealmRoleAssignedToUser(ctx context.Context, kcUserId, roleName string) (bool, error) {
	return false, nil
}
func (m *mockKeycloakService) IssueWorkspaceTicket(ctx context.Context, kcUserId string, workspaceID uint) (string, map[string]interface{}, error) {
	return "", nil, nil
}
func (m *mockKeycloakService) SetupPredefinedRoles(ctx context.Context, accessToken string) error {
	return nil
}
func (m *mockKeycloakService) GetClientCredentialsToken(ctx context.Context) (*gocloak.JWT, error) {
	return nil, nil
}
func (m *mockKeycloakService) CreatePendingUser(ctx context.Context, req *model.SignupRequest) (string, error) {
	return "", nil
}
func (m *mockKeycloakService) ResetPassword(ctx context.Context, kcUserID, newPassword string) error {
	return nil
}
func (m *mockKeycloakService) AddRealmRoleToGroup(ctx context.Context, groupName, roleName string) error {
	return nil
}
func (m *mockKeycloakService) RemoveRealmRoleFromGroup(ctx context.Context, groupName, roleName string) error {
	return nil
}
func (m *mockKeycloakService) CheckSAMLClientConfig(ctx context.Context, clientID string) (string, error) {
	return "", nil
}
