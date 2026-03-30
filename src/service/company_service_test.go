package service

import (
	"os"
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Company{}))
	return db
}

func seedCompany(t *testing.T, db *gorm.DB, name, realm, status string) *model.Company {
	repo := repository.NewCompanyRepository(db)
	c := &model.Company{Name: name, RealmName: realm, Status: status}
	require.NoError(t, repo.Create(c))
	return c
}

// TestCompanyService_CreateCompany_Conflict realm_name 중복 시 CONFLICT 에러 반환
func TestCompanyService_CreateCompany_Conflict(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Existing", "duplicate-realm", "active")

	req := &model.CompanyRequest{
		Name:           "New Company",
		RealmName:      "duplicate-realm",
		KcClientID:     "client-id",
		KcClientSecret: "secret",
	}

	_, err := svc.CreateCompany(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CONFLICT:")
}

// TestCompanyService_CreateCompany_RealmError KC 미설정 시 REALM_ERROR 반환
func TestCompanyService_CreateCompany_RealmError(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	req := &model.CompanyRequest{
		Name:           "New Company",
		RealmName:      "new-realm",
		KcClientID:     "client-id",
		KcClientSecret: "secret",
	}

	_, err := svc.CreateCompany(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REALM_ERROR:")
}

// TestCompanyService_GetCompany_NotFound 회사 없을 때 not found 에러
func TestCompanyService_GetCompany_NotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	_, err := svc.GetCompany()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestCompanyService_GetCompany 회사 조회 성공
func TestCompanyService_GetCompany(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "My Company", "my-realm", "active")

	resp, err := svc.GetCompany()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "My Company", resp.Name)
	assert.Equal(t, "my-realm", resp.RealmName)
	assert.Equal(t, "active", resp.Status)
}

// TestCompanyService_GetCompany_SecretExcluded kc_client_secret이 응답에서 제외됨
func TestCompanyService_GetCompany_SecretExcluded(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewCompanyRepository(db)
	svc := NewCompanyService(db)

	c := &model.Company{Name: "C", RealmName: "r", KcClientSecret: "super-secret", Status: "active"}
	require.NoError(t, repo.Create(c))

	resp, err := svc.GetCompany()
	require.NoError(t, err)
	// CompanyResponse 구조체에 KcClientSecret 필드가 없음 — 타입 수준에서 강제됨
	assert.NotNil(t, resp)
	assert.Equal(t, "C", resp.Name)
}

// TestCompanyService_UpdateCompany 이름/설명 수정
func TestCompanyService_UpdateCompany(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Original", "realm", "active")

	req := &model.CompanyUpdateRequest{Name: "Updated Name", Description: "New description"}
	resp, err := svc.UpdateCompany(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Updated Name", resp.Name)
	assert.Equal(t, "New description", resp.Description)
}

// TestCompanyService_UpdateCompany_NotFound 회사 없을 때 에러
func TestCompanyService_UpdateCompany_NotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	_, err := svc.UpdateCompany(&model.CompanyUpdateRequest{Name: "X"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestCompanyService_DeactivateCompany 비활성화 처리
func TestCompanyService_DeactivateCompany(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Company", "realm", "active")

	resp, err := svc.DeactivateCompany()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "inactive", resp.Status)
}

// TestCompanyService_DeactivateCompany_Idempotent 이미 inactive인 경우 멱등 처리
func TestCompanyService_DeactivateCompany_Idempotent(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Company", "realm", "inactive")

	resp, err := svc.DeactivateCompany()
	require.NoError(t, err)
	assert.Equal(t, "inactive", resp.Status)
}

// TestCompanyService_ActivateCompany 활성화 처리
func TestCompanyService_ActivateCompany(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Company", "realm", "inactive")

	resp, err := svc.ActivateCompany()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "active", resp.Status)
}

// TestCompanyService_ActivateCompany_Idempotent 이미 active인 경우 멱등 처리
func TestCompanyService_ActivateCompany_Idempotent(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Company", "realm", "active")

	resp, err := svc.ActivateCompany()
	require.NoError(t, err)
	assert.Equal(t, "active", resp.Status)
}

// TestCompanyService_CreateDefaultCompany 기본 회사 생성
func TestCompanyService_CreateDefaultCompany(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	os.Setenv("MC_IAM_MANAGER_COMPANY_NAME", "Test Default Company")
	os.Setenv("MC_IAM_MANAGER_KEYCLOAK_REALM", "test-realm")
	os.Setenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME", "test-client")
	os.Setenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("MC_IAM_MANAGER_COMPANY_NAME")
		os.Unsetenv("MC_IAM_MANAGER_KEYCLOAK_REALM")
		os.Unsetenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME")
		os.Unsetenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET")
	}()

	err := svc.CreateDefaultCompany()
	require.NoError(t, err)

	resp, err := svc.GetCompany()
	require.NoError(t, err)
	assert.Equal(t, "Test Default Company", resp.Name)
	assert.Equal(t, "test-realm", resp.RealmName)
}

// TestCompanyService_CreateDefaultCompany_DefaultName env 미설정 시 "Default Company"
func TestCompanyService_CreateDefaultCompany_DefaultName(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	os.Unsetenv("MC_IAM_MANAGER_COMPANY_NAME")

	err := svc.CreateDefaultCompany()
	require.NoError(t, err)

	resp, err := svc.GetCompany()
	require.NoError(t, err)
	assert.Equal(t, "Default Company", resp.Name)
}

// TestCompanyService_CreateDefaultCompany_Idempotent 이미 회사 있으면 skip
func TestCompanyService_CreateDefaultCompany_Idempotent(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCompanyService(db)

	seedCompany(t, db, "Existing", "realm", "active")

	err := svc.CreateDefaultCompany()
	require.NoError(t, err)

	// 여전히 1개만 존재해야 함
	repo := repository.NewCompanyRepository(db)
	count, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
