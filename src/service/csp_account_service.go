package service

import (
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CspAccountService CSP 계정 서비스
type CspAccountService struct {
	db              *gorm.DB
	cspAccountRepo  *repository.CspAccountRepository
	cspIdpConfigRepo *repository.CspIdpConfigRepository
	cspPolicyRepo    *repository.CspPolicyRepository
}

// NewCspAccountService 새 CspAccountService 인스턴스 생성
func NewCspAccountService(db *gorm.DB) *CspAccountService {
	return &CspAccountService{
		db:              db,
		cspAccountRepo:  repository.NewCspAccountRepository(db),
		cspIdpConfigRepo: repository.NewCspIdpConfigRepository(db),
		cspPolicyRepo:    repository.NewCspPolicyRepository(db),
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

// ValidateCspAccount CSP 계정 유효성 검증
func (s *CspAccountService) ValidateCspAccount(id uint) error {
	account, err := s.cspAccountRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("CSP account not found with ID: %d", id)
	}

	// CSP 타입별 필수 필드 검증
	switch account.CspType {
	case "aws":
		if account.GetAccountID() == "" {
			return fmt.Errorf("AWS account_id is required")
		}
	case "gcp":
		if account.GetProjectID() == "" {
			return fmt.Errorf("GCP project_id is required")
		}
	case "azure":
		if account.GetSubscriptionID() == "" {
			return fmt.Errorf("Azure subscription_id is required")
		}
		if account.GetTenantID() == "" {
			return fmt.Errorf("Azure tenant_id is required")
		}
	default:
		return fmt.Errorf("unsupported CSP type: %s", account.CspType)
	}

	log.Printf("Validated CSP account: %s (ID: %d)", account.Name, account.ID)
	return nil
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
