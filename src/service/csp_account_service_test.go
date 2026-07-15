package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.CspAccount{},
		&model.CspIdpConfig{},
		&model.CspPolicy{},
		&model.CspRolePolicyMapping{},
		&model.CspRole{},
	)
	require.NoError(t, err)

	return db
}

func newTestService(t *testing.T) (*CspAccountService, *gorm.DB) {
	db := setupTestDB(t)
	return NewCspAccountService(db), db
}

// TestValidateCspAccount_AWS_Valid: AWS 계정에 account_id가 있으면 검증 통과
func TestCspAccountValidate_AWS_Valid(t *testing.T) {
	svc, _ := newTestService(t)

	account := &model.CspAccount{
		Name:    "test-aws",
		CspType: "aws",
		AccountInfo: map[string]string{
			"account_id": "123456789012",
			"region":     "ap-northeast-2",
		},
		IsActive: true,
	}
	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:        account.Name,
		CspType:     account.CspType,
		AccountInfo: account.AccountInfo,
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.NoError(t, err)
}

// TestValidateCspAccount_AWS_MissingAccountID: AWS 계정에 account_id가 없으면 에러
func TestCspAccountValidate_AWS_MissingAccountID(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:        "test-aws-missing",
		CspType:     "aws",
		AccountInfo: map[string]string{"region": "us-east-1"},
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AWS account_id is required")
}

// TestValidateCspAccount_GCP_Valid: GCP 계정에 project_id가 있으면 검증 통과
func TestCspAccountValidate_GCP_Valid(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:    "test-gcp",
		CspType: "gcp",
		AccountInfo: map[string]string{
			"project_id": "my-gcp-project",
		},
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.NoError(t, err)
}

// TestValidateCspAccount_GCP_MissingProjectID: GCP 계정에 project_id가 없으면 에러
func TestCspAccountValidate_GCP_MissingProjectID(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:        "test-gcp-missing",
		CspType:     "gcp",
		AccountInfo: map[string]string{},
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GCP project_id is required")
}

// TestValidateCspAccount_Azure_Valid: Azure 계정에 subscription_id와 tenant_id가 있으면 통과
func TestCspAccountValidate_Azure_Valid(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:    "test-azure",
		CspType: "azure",
		AccountInfo: map[string]string{
			"subscription_id": "sub-001",
			"tenant_id":       "tenant-001",
		},
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.NoError(t, err)
}

// TestValidateCspAccount_Azure_MissingSubscriptionID: Azure 계정에 subscription_id가 없으면 에러
func TestCspAccountValidate_Azure_MissingSubscriptionID(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:    "test-azure-no-sub",
		CspType: "azure",
		AccountInfo: map[string]string{
			"tenant_id": "tenant-001",
		},
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Azure subscription_id is required")
}

// TestValidateCspAccount_Azure_MissingTenantID: Azure 계정에 tenant_id가 없으면 에러
func TestCspAccountValidate_Azure_MissingTenantID(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:    "test-azure-no-tenant",
		CspType: "azure",
		AccountInfo: map[string]string{
			"subscription_id": "sub-001",
		},
	})
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Azure tenant_id is required")
}

// TestValidateCspAccount_UnsupportedType: 지원하지 않는 CSP 타입은 에러
func TestCspAccountValidate_UnsupportedType(t *testing.T) {
	svc, db := newTestService(t)

	// CspType 검증이 서비스에 없으므로 DB에 직접 삽입
	account := &model.CspAccount{
		Name:    "test-unknown",
		CspType: "unknown",
		AccountInfo: map[string]string{
			"some_field": "some_value",
		},
		IsActive: true,
	}
	err := db.Create(account).Error
	require.NoError(t, err)

	_, err = svc.ValidateCspAccount(context.Background(), account.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported CSP type")
}

// TestValidateCspAccount_NotFound: 존재하지 않는 계정은 에러
func TestCspAccountValidate_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.ValidateCspAccount(context.Background(), 9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CSP account not found with ID: 9999")
}

// TestCreateCspAccount_DuplicateName: 같은 이름+타입 중복 생성 불가
func TestCspAccountCreate_DuplicateName(t *testing.T) {
	svc, _ := newTestService(t)

	req := &model.CreateCspAccountRequest{
		Name:        "dup-account",
		CspType:     "aws",
		AccountInfo: map[string]string{"account_id": "111"},
	}
	_, err := svc.CreateCspAccount(req)
	require.NoError(t, err)

	_, err = svc.CreateCspAccount(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestDeleteCspAccount_WithIdpConfig: IDP 설정이 있으면 삭제 불가
func TestCspAccountDelete_WithIdpConfig(t *testing.T) {
	svc, db := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:        "del-with-idp",
		CspType:     "aws",
		AccountInfo: map[string]string{"account_id": "999"},
	})
	require.NoError(t, err)

	// IDP 설정 삽입
	idpConfig := &model.CspIdpConfig{
		Name:         "test-idp",
		CspAccountID: created.ID,
		AuthMethod:   model.AuthMethodOIDC,
		Config:       map[string]string{"role_arn": "arn:aws:iam::999:role/test"},
		IsActive:     true,
	}
	err = db.Create(idpConfig).Error
	require.NoError(t, err)

	err = svc.DeleteCspAccount(created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IDP configs are associated")
}

// TestDeleteCspAccount_WithPolicy: 정책이 있으면 삭제 불가
func TestCspAccountDelete_WithPolicy(t *testing.T) {
	svc, db := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:        "del-with-policy",
		CspType:     "aws",
		AccountInfo: map[string]string{"account_id": "888"},
	})
	require.NoError(t, err)

	policy := &model.CspPolicy{
		Name:         "test-policy",
		CspAccountID: created.ID,
		PolicyType:   model.PolicyTypeManaged,
	}
	err = db.Create(policy).Error
	require.NoError(t, err)

	err = svc.DeleteCspAccount(created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policies are associated")
}

// TestDeleteCspAccount_Success: IDP/정책 없이 삭제 성공
func TestCspAccountDelete_Success(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:        "del-success",
		CspType:     "aws",
		AccountInfo: map[string]string{"account_id": "777"},
	})
	require.NoError(t, err)

	err = svc.DeleteCspAccount(created.ID)
	assert.NoError(t, err)

	// 삭제 후 조회 시 에러 확인
	_, err = svc.ValidateCspAccount(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("CSP account not found with ID: %d", created.ID))
}

// TestUpdateCspAccount_DuplicateName: 수정 시 다른 계정과 이름 충돌 에러
func TestCspAccountUpdate_DuplicateName(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:    "account-a",
		CspType: "aws",
		AccountInfo: map[string]string{"account_id": "001"},
	})
	require.NoError(t, err)

	b, err := svc.CreateCspAccount(&model.CreateCspAccountRequest{
		Name:    "account-b",
		CspType: "aws",
		AccountInfo: map[string]string{"account_id": "002"},
	})
	require.NoError(t, err)

	// account-b를 account-a 이름으로 변경 시도
	_, err = svc.UpdateCspAccount(b.ID, &model.UpdateCspAccountRequest{Name: "account-a"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
