package service

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CompanyService 회사 정보 서비스
type CompanyService struct {
	db              *gorm.DB
	companyRepo     *repository.CompanyRepository
	keycloakService KeycloakService
}

// NewCompanyService 새 CompanyService 인스턴스 생성
func NewCompanyService(db *gorm.DB) *CompanyService {
	return &CompanyService{
		db:              db,
		companyRepo:     repository.NewCompanyRepository(db),
		keycloakService: NewKeycloakService(),
	}
}

// CreateCompany 회사 생성 (COMP-001)
func (s *CompanyService) CreateCompany(req *model.CompanyRequest) (*model.CompanyResponse, error) {
	exists, err := s.companyRepo.ExistsByRealmName(req.RealmName)
	if err != nil {
		return nil, fmt.Errorf("failed to check realm_name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("CONFLICT: company with realm_name '%s' already exists", req.RealmName)
	}

	if err := s.ensureRealm(req.RealmName); err != nil {
		return nil, fmt.Errorf("REALM_ERROR: %w", err)
	}

	company := &model.Company{
		Name:           req.Name,
		Description:    req.Description,
		RealmName:      req.RealmName,
		KcClientID:     req.KcClientID,
		KcClientSecret: req.KcClientSecret,
		Status:         "active",
	}

	if err := s.companyRepo.Create(company); err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	log.Printf("[INFO] Company created: name=%s, realm_name=%s", company.Name, company.RealmName)
	return company.ToResponse(), nil
}

// GetCompany 회사 조회 (COMP-002, 싱글톤)
func (s *CompanyService) GetCompany() (*model.CompanyResponse, error) {
	company, err := s.companyRepo.First()
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found")
	}
	return company.ToResponse(), nil
}

// UpdateCompany 회사 수정 (COMP-003, name/description만 변경 가능)
func (s *CompanyService) UpdateCompany(req *model.CompanyUpdateRequest) (*model.CompanyResponse, error) {
	company, err := s.companyRepo.First()
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found")
	}

	company.Name = req.Name
	company.Description = req.Description

	if err := s.companyRepo.Save(company); err != nil {
		return nil, fmt.Errorf("failed to update company: %w", err)
	}

	log.Printf("[INFO] Company updated: id=%d, name=%s", company.ID, company.Name)
	return company.ToResponse(), nil
}

// DeactivateCompany 회사 비활성화 (COMP-004, 멱등 처리)
func (s *CompanyService) DeactivateCompany() (*model.CompanyResponse, error) {
	company, err := s.companyRepo.First()
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found")
	}

	company.Status = "inactive"
	if err := s.companyRepo.Save(company); err != nil {
		return nil, fmt.Errorf("failed to deactivate company: %w", err)
	}

	log.Printf("[INFO] Company deactivated: id=%d", company.ID)
	return company.ToResponse(), nil
}

// ActivateCompany 회사 활성화 (COMP-005, 멱등 처리)
func (s *CompanyService) ActivateCompany() (*model.CompanyResponse, error) {
	company, err := s.companyRepo.First()
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found")
	}

	company.Status = "active"
	if err := s.companyRepo.Save(company); err != nil {
		return nil, fmt.Errorf("failed to activate company: %w", err)
	}

	log.Printf("[INFO] Company activated: id=%d", company.ID)
	return company.ToResponse(), nil
}

// CreateDefaultCompany initial-admin 시 기본 회사 자동 생성 (COMP-006)
// Count()>0이면 skip (멱등), 실패 시 WARNING만 (non-fatal)
func (s *CompanyService) CreateDefaultCompany() error {
	count, err := s.companyRepo.Count()
	if err != nil {
		return fmt.Errorf("failed to count companies: %w", err)
	}
	if count > 0 {
		log.Printf("[INFO] Default company already exists, skipping creation")
		return nil
	}

	companyName := os.Getenv("MC_IAM_MANAGER_COMPANY_NAME")
	if companyName == "" {
		companyName = "Default Company"
	}
	realmName := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_REALM")
	kcClientID := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME")
	kcClientSecret := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET")

	company := &model.Company{
		Name:           companyName,
		RealmName:      realmName,
		KcClientID:     kcClientID,
		KcClientSecret: kcClientSecret,
		Status:         "active",
	}

	if err := s.companyRepo.Create(company); err != nil {
		return fmt.Errorf("failed to create default company: %w", err)
	}

	log.Printf("[INFO] Default company created: name=%s", companyName)
	return nil
}

// ensureRealm Keycloak에 realm이 없으면 생성
func (s *CompanyService) ensureRealm(realmName string) error {
	if config.KC == nil {
		return fmt.Errorf("keycloak not configured")
	}

	adminToken, err := s.keycloakService.KeycloakAdminLogin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	_, err = config.KC.Client.GetRealm(context.Background(), adminToken.AccessToken, realmName)
	if err == nil {
		log.Printf("[INFO] Realm '%s' already exists in Keycloak, using existing realm", realmName)
		return nil
	}

	enabled := true
	_, err = config.KC.Client.CreateRealm(context.Background(), adminToken.AccessToken, gocloak.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	if err != nil {
		return fmt.Errorf("failed to create realm '%s': %w", realmName, err)
	}

	log.Printf("[INFO] Realm '%s' created in Keycloak", realmName)
	return nil
}
