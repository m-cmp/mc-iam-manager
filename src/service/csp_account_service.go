package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CspAccountService CSP 계정 서비스
type CspAccountService struct {
	db               *gorm.DB
	cspAccountRepo   *repository.CspAccountRepository
	cspIdpConfigRepo *repository.CspIdpConfigRepository
	cspPolicyRepo    *repository.CspPolicyRepository
	cspRoleRepo      *repository.CspRoleRepository
	awsCredService   AwsCredentialService
}

// NewCspAccountService 새 CspAccountService 인스턴스 생성
func NewCspAccountService(db *gorm.DB) *CspAccountService {
	return &CspAccountService{
		db:               db,
		cspAccountRepo:   repository.NewCspAccountRepository(db),
		cspIdpConfigRepo: repository.NewCspIdpConfigRepository(db),
		cspPolicyRepo:    repository.NewCspPolicyRepository(db),
		cspRoleRepo:      repository.NewCspRoleRepository(db),
		awsCredService:   NewAwsCredentialService(),
	}
}

// CreateCspAccount CSP 계정 생성
func (s *CspAccountService) CreateCspAccount(req *model.CreateCspAccountRequest) (*model.CspAccount, error) {
	// 이름 중복 확인
	exists, err := s.cspAccountRepo.ExistsByNameAndCspType(req.Name, req.CspType)
	if err != nil {
		return nil, fmt.Errorf("failed to check CSP account existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("CSP account with name '%s' and type '%s' already exists", req.Name, req.CspType)
	}

	// CSP 계정 생성
	account := &model.CspAccount{
		Name:        req.Name,
		CspType:     req.CspType,
		AccountInfo: req.AccountInfo,
		IsActive:    true,
		Description: req.Description,
	}

	if err := s.cspAccountRepo.Create(account); err != nil {
		return nil, fmt.Errorf("failed to create CSP account: %w", err)
	}

	log.Printf("Created CSP account: %s (type: %s)", account.Name, account.CspType)
	return account, nil
}

// GetCspAccountByID ID로 CSP 계정 조회
func (s *CspAccountService) GetCspAccountByID(id uint) (*model.CspAccount, error) {
	account, err := s.cspAccountRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("CSP account not found with ID: %d", id)
	}
	return account, nil
}

// GetCspAccountByName 이름으로 CSP 계정 조회
func (s *CspAccountService) GetCspAccountByName(name string) (*model.CspAccount, error) {
	account, err := s.cspAccountRepo.GetByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	return account, nil
}

// ListCspAccounts CSP 계정 목록 조회
func (s *CspAccountService) ListCspAccounts(filter *model.CspAccountFilter) ([]*model.CspAccount, error) {
	accounts, err := s.cspAccountRepo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list CSP accounts: %w", err)
	}
	return accounts, nil
}

// UpdateCspAccount CSP 계정 수정
func (s *CspAccountService) UpdateCspAccount(id uint, req *model.UpdateCspAccountRequest) (*model.CspAccount, error) {
	// 기존 계정 조회
	account, err := s.cspAccountRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("CSP account not found with ID: %d", id)
	}

	// 필드 업데이트
	if req.Name != "" {
		// 이름 변경 시 중복 확인
		if req.Name != account.Name {
			exists, err := s.cspAccountRepo.ExistsByNameAndCspType(req.Name, account.CspType)
			if err != nil {
				return nil, fmt.Errorf("failed to check CSP account existence: %w", err)
			}
			if exists {
				return nil, fmt.Errorf("CSP account with name '%s' already exists", req.Name)
			}
		}
		account.Name = req.Name
	}
	if req.AccountInfo != nil {
		account.AccountInfo = req.AccountInfo
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}
	if req.Description != "" {
		account.Description = req.Description
	}

	if err := s.cspAccountRepo.Update(account); err != nil {
		return nil, fmt.Errorf("failed to update CSP account: %w", err)
	}

	log.Printf("Updated CSP account: %s (ID: %d)", account.Name, account.ID)
	return account, nil
}

// DeleteCspAccount CSP 계정 삭제
func (s *CspAccountService) DeleteCspAccount(id uint) error {
	// 계정 존재 확인
	exists, err := s.cspAccountRepo.ExistsByID(id)
	if err != nil {
		return fmt.Errorf("failed to check CSP account existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("CSP account not found with ID: %d", id)
	}

	// 연관된 IDP 설정 확인
	idpCount, err := s.cspIdpConfigRepo.CountByAccountID(id)
	if err != nil {
		return fmt.Errorf("failed to count IDP configs: %w", err)
	}
	if idpCount > 0 {
		return fmt.Errorf("cannot delete CSP account: %d IDP configs are associated", idpCount)
	}

	// 연관된 정책 확인
	policyCount, err := s.cspPolicyRepo.CountByAccountID(id)
	if err != nil {
		return fmt.Errorf("failed to count policies: %w", err)
	}
	if policyCount > 0 {
		return fmt.Errorf("cannot delete CSP account: %d policies are associated", policyCount)
	}

	if err := s.cspAccountRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete CSP account: %w", err)
	}

	log.Printf("Deleted CSP account with ID: %d", id)
	return nil
}

// ValidateCspAccount CSP 계정 인프라 검증 — 연결된 CspRole의 IDP/IAM 설정을 실제 CSP API로 확인
func (s *CspAccountService) ValidateCspAccount(ctx context.Context, id uint) (*model.CspAccountValidationResponse, error) {
	account, err := s.cspAccountRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("CSP account not found with ID: %d", id)
	}

	roles, err := s.cspRoleRepo.GetByCspAccountID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP roles: %w", err)
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("no CSP roles found for account ID: %d", id)
	}

	resp := &model.CspAccountValidationResponse{
		AccountID:   account.ID,
		AccountName: account.Name,
		CspType:     account.CspType,
		Results:     make([]model.CspAccountValidationResult, 0, len(roles)),
	}

	for _, role := range roles {
		result := s.validateCspRole(ctx, role)
		resp.Results = append(resp.Results, result)
	}

	log.Printf("Validated CSP account: %s (ID: %d), roles: %d", account.Name, account.ID, len(roles))
	return resp, nil
}

// validateCspRole CspRole 단위로 CSP 인프라 검증
func (s *CspAccountService) validateCspRole(ctx context.Context, role *model.CspRole) model.CspAccountValidationResult {
	result := model.CspAccountValidationResult{
		CspRoleID:   role.ID,
		CspRoleName: role.Name,
		CspType:     role.CspType,
	}

	if role.CspIdpConfig == nil {
		result.AuthMethod = "UNKNOWN"
		result.Valid = false
		result.Error = "CspIdpConfig not linked to this CspRole"
		return result
	}

	result.AuthMethod = string(role.CspIdpConfig.AuthMethod)

	valCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch role.CspType {
	case "aws":
		s.validateAWSRole(valCtx, role, &result)
	default:
		s.validateGenericRole(valCtx, role, &result)
	}

	return result
}

// validateAWSRole AWS CspRole 검증 (OIDC/SAML: IDP+RoleTrust, SECRET_KEY: CallerIdentity)
func (s *CspAccountService) validateAWSRole(ctx context.Context, role *model.CspRole, result *model.CspAccountValidationResult) {
	cfg := role.CspIdpConfig
	idpIdentifier := role.IdpIdentifier
	iamIdentifier := role.IamIdentifier

	switch cfg.AuthMethod {
	case model.AuthMethodOIDC:
		if idpIdentifier == "" {
			result.Valid = false
			result.Error = "IdpIdentifier (OIDC Provider ARN) is empty"
			return
		}
		detail, err := s.awsCredService.CheckOIDCProvider(ctx, idpIdentifier)
		if err != nil {
			result.Valid = false
			result.Error = fmt.Sprintf("OIDC provider check failed: %v", err)
			return
		}
		if iamIdentifier != "" {
			trustDetail, err := s.awsCredService.CheckRoleTrust(ctx, iamIdentifier, "sts:AssumeRoleWithWebIdentity", idpIdentifier)
			if err != nil {
				result.Valid = false
				result.Error = fmt.Sprintf("Role trust check failed: %v", err)
				return
			}
			detail += " | " + trustDetail
		}
		result.Valid = true
		result.Detail = detail

	case model.AuthMethodSAML:
		if idpIdentifier == "" {
			result.Valid = false
			result.Error = "IdpIdentifier (SAML Provider ARN) is empty"
			return
		}
		detail, err := s.awsCredService.CheckSAMLProvider(ctx, idpIdentifier)
		if err != nil {
			result.Valid = false
			result.Error = fmt.Sprintf("SAML provider check failed: %v", err)
			return
		}
		if iamIdentifier != "" {
			trustDetail, err := s.awsCredService.CheckRoleTrust(ctx, iamIdentifier, "sts:AssumeRoleWithSAML", idpIdentifier)
			if err != nil {
				result.Valid = false
				result.Error = fmt.Sprintf("Role trust check failed: %v", err)
				return
			}
			detail += " | " + trustDetail
		}
		result.Valid = true
		result.Detail = detail

	case model.AuthMethodSecretKey:
		accessKeyID := cfg.GetAccessKeyID()
		secretKey := cfg.GetSecretAccessKey()
		if accessKeyID == "" || secretKey == "" {
			result.Valid = false
			result.Error = "access_key_id or secret_access_key is empty in CspIdpConfig"
			return
		}
		detail, err := s.awsCredService.CheckCallerIdentity(ctx, accessKeyID, secretKey)
		if err != nil {
			result.Valid = false
			result.Error = fmt.Sprintf("CallerIdentity check failed: %v", err)
			return
		}
		result.Valid = true
		result.Detail = detail

	default:
		result.Valid = false
		result.Error = fmt.Sprintf("unsupported auth method for AWS: %s", cfg.AuthMethod)
	}
}

// validateGenericRole GCP/Azure 등 — config 필수 필드 존재 확인 (인프라 레벨 체크 미구현)
func (s *CspAccountService) validateGenericRole(ctx context.Context, role *model.CspRole, result *model.CspAccountValidationResult) {
	cfg := role.CspIdpConfig
	if cfg.AuthMethod == model.AuthMethodSecretKey {
		if cfg.GetAccessKeyID() == "" || cfg.GetSecretAccessKey() == "" {
			result.Valid = false
			result.Error = "access_key_id or secret_access_key is empty in CspIdpConfig"
			return
		}
		result.Valid = true
		result.Detail = "SECRET_KEY config fields present (live validation not implemented for this CSP)"
		return
	}
	// OIDC/SAML: IdpIdentifier + IamIdentifier 존재 확인
	if role.IdpIdentifier == "" {
		result.Valid = false
		result.Error = "IdpIdentifier is empty"
		return
	}
	result.Valid = true
	result.Detail = fmt.Sprintf("%s config fields present (live infrastructure check not implemented for %s)", cfg.AuthMethod, role.CspType)
}

// GetActiveCspAccounts 활성 CSP 계정 목록 조회
func (s *CspAccountService) GetActiveCspAccounts() ([]*model.CspAccount, error) {
	accounts, err := s.cspAccountRepo.GetActiveAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get active CSP accounts: %w", err)
	}
	return accounts, nil
}

// GetCspAccountsByCspType CSP 타입별 계정 목록 조회
func (s *CspAccountService) GetCspAccountsByCspType(cspType string) ([]*model.CspAccount, error) {
	accounts, err := s.cspAccountRepo.GetByCspType(cspType)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP accounts by type: %w", err)
	}
	return accounts, nil
}

// ActivateCspAccount CSP 계정 활성화
func (s *CspAccountService) ActivateCspAccount(id uint) error {
	account, err := s.cspAccountRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("CSP account not found with ID: %d", id)
	}

	account.IsActive = true
	if err := s.cspAccountRepo.Update(account); err != nil {
		return fmt.Errorf("failed to activate CSP account: %w", err)
	}

	log.Printf("Activated CSP account: %s (ID: %d)", account.Name, account.ID)
	return nil
}

// DeactivateCspAccount CSP 계정 비활성화
func (s *CspAccountService) DeactivateCspAccount(id uint) error {
	account, err := s.cspAccountRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("CSP account not found with ID: %d", id)
	}

	account.IsActive = false
	if err := s.cspAccountRepo.Update(account); err != nil {
		return fmt.Errorf("failed to deactivate CSP account: %w", err)
	}

	log.Printf("Deactivated CSP account: %s (ID: %d)", account.Name, account.ID)
	return nil
}
