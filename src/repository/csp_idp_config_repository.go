package repository

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// CspIdpConfigRepository CSP IDP 설정 레포지토리
type CspIdpConfigRepository struct {
	db *gorm.DB
}

// NewCspIdpConfigRepository 새 CspIdpConfigRepository 인스턴스 생성
func NewCspIdpConfigRepository(db *gorm.DB) *CspIdpConfigRepository {
	return &CspIdpConfigRepository{db: db}
}

// Create CSP IDP 설정 생성
func (r *CspIdpConfigRepository) Create(config *model.CspIdpConfig) error {
	if err := r.db.Create(config).Error; err != nil {
		return fmt.Errorf("failed to create CSP IDP config: %w", err)
	}
	return nil
}

// GetByID ID로 CSP IDP 설정 조회
func (r *CspIdpConfigRepository) GetByID(id uint) (*model.CspIdpConfig, error) {
	var config model.CspIdpConfig
	if err := r.db.Preload("CspAccount").First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP IDP config by ID: %w", err)
	}
	return &config, nil
}

// GetByName 이름으로 CSP IDP 설정 조회
func (r *CspIdpConfigRepository) GetByName(name string) (*model.CspIdpConfig, error) {
	var config model.CspIdpConfig
	if err := r.db.Preload("CspAccount").Where("name = ?", name).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP IDP config by name: %w", err)
	}
	return &config, nil
}

// GetByNameAndAccountID 이름과 계정 ID로 CSP IDP 설정 조회
func (r *CspIdpConfigRepository) GetByNameAndAccountID(name string, accountID uint) (*model.CspIdpConfig, error) {
	var config model.CspIdpConfig
	if err := r.db.Preload("CspAccount").Where("name = ? AND csp_account_id = ?", name, accountID).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP IDP config: %w", err)
	}
	return &config, nil
}

// List CSP IDP 설정 목록 조회
func (r *CspIdpConfigRepository) List(filter *model.CspIdpConfigFilter) ([]*model.CspIdpConfig, error) {
	var configs []*model.CspIdpConfig
	query := r.db.Model(&model.CspIdpConfig{}).Preload("CspAccount")

	if filter != nil {
		if filter.CspAccountID != nil {
			query = query.Where("csp_account_id = ?", *filter.CspAccountID)
		}
		if filter.AuthMethod != "" {
			query = query.Where("auth_method = ?", filter.AuthMethod)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
	}

	if err := query.Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to list CSP IDP configs: %w", err)
	}
	return configs, nil
}

// Update CSP IDP 설정 수정
func (r *CspIdpConfigRepository) Update(config *model.CspIdpConfig) error {
	if err := r.db.Save(config).Error; err != nil {
		return fmt.Errorf("failed to update CSP IDP config: %w", err)
	}
	return nil
}

// Delete CSP IDP 설정 삭제
func (r *CspIdpConfigRepository) Delete(id uint) error {
	result := r.db.Delete(&model.CspIdpConfig{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete CSP IDP config: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("CSP IDP config not found")
	}
	return nil
}

// ExistsByID ID로 CSP IDP 설정 존재 여부 확인
func (r *CspIdpConfigRepository) ExistsByID(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspIdpConfig{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP IDP config existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByName 이름으로 CSP IDP 설정 존재 여부 확인
func (r *CspIdpConfigRepository) ExistsByName(name string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspIdpConfig{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP IDP config existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameAndAccountID 이름과 계정 ID로 CSP IDP 설정 존재 여부 확인
func (r *CspIdpConfigRepository) ExistsByNameAndAccountID(name string, accountID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspIdpConfig{}).Where("name = ? AND csp_account_id = ?", name, accountID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP IDP config existence: %w", err)
	}
	return count > 0, nil
}

// GetByAccountID 계정 ID로 CSP IDP 설정 목록 조회
func (r *CspIdpConfigRepository) GetByAccountID(accountID uint) ([]*model.CspIdpConfig, error) {
	var configs []*model.CspIdpConfig
	if err := r.db.Preload("CspAccount").Where("csp_account_id = ?", accountID).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP IDP configs by account ID: %w", err)
	}
	return configs, nil
}

// GetActiveConfigs 활성 CSP IDP 설정 목록 조회
func (r *CspIdpConfigRepository) GetActiveConfigs() ([]*model.CspIdpConfig, error) {
	var configs []*model.CspIdpConfig
	if err := r.db.Preload("CspAccount").Where("is_active = ?", true).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get active CSP IDP configs: %w", err)
	}
	return configs, nil
}

// GetActiveByAccountID 특정 계정의 활성 CSP IDP 설정 목록 조회
func (r *CspIdpConfigRepository) GetActiveByAccountID(accountID uint) ([]*model.CspIdpConfig, error) {
	var configs []*model.CspIdpConfig
	if err := r.db.Preload("CspAccount").Where("csp_account_id = ? AND is_active = ?", accountID, true).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get active CSP IDP configs by account ID: %w", err)
	}
	return configs, nil
}

// GetByAuthMethod 인증 방식으로 CSP IDP 설정 목록 조회
func (r *CspIdpConfigRepository) GetByAuthMethod(authMethod model.AuthMethodType) ([]*model.CspIdpConfig, error) {
	var configs []*model.CspIdpConfig
	if err := r.db.Preload("CspAccount").Where("auth_method = ?", authMethod).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP IDP configs by auth method: %w", err)
	}
	return configs, nil
}

// CountByAccountID 특정 계정의 CSP IDP 설정 개수 조회
func (r *CspIdpConfigRepository) CountByAccountID(accountID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.CspIdpConfig{}).Where("csp_account_id = ?", accountID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count CSP IDP configs: %w", err)
	}
	return count, nil
}

// CspIdpSummaryRow GetSummary 집계 결과 행
type CspIdpSummaryRow struct {
	CspAccountID   uint   `gorm:"column:csp_account_id"`
	CspAccountName string `gorm:"column:csp_account_name"`
	CspType        string `gorm:"column:csp_type"`
	TotalCount     int    `gorm:"column:total_count"`
	ActiveCount    int    `gorm:"column:active_count"`
	OidcCount      int    `gorm:"column:oidc_count"`
	SamlCount      int    `gorm:"column:saml_count"`
	SecretKeyCount int    `gorm:"column:secret_key_count"`
}

// GetSummary CSP 계정별 IDP 설정 현황 집계 조회
func (r *CspIdpConfigRepository) GetSummary() ([]CspIdpSummaryRow, error) {
	var rows []CspIdpSummaryRow
	sql := `
		SELECT
			c.id AS csp_account_id,
			c.name AS csp_account_name,
			c.csp_type,
			COUNT(i.id) AS total_count,
			COUNT(CASE WHEN i.is_active THEN 1 END) AS active_count,
			COUNT(CASE WHEN i.auth_method = 'OIDC' THEN 1 END) AS oidc_count,
			COUNT(CASE WHEN i.auth_method = 'SAML' THEN 1 END) AS saml_count,
			COUNT(CASE WHEN i.auth_method = 'SECRET_KEY' THEN 1 END) AS secret_key_count
		FROM mcmp_csp_accounts c
		LEFT JOIN mcmp_csp_idp_configs i ON i.csp_account_id = c.id
		GROUP BY c.id, c.name, c.csp_type
		ORDER BY c.id
	`
	if err := r.db.Raw(sql).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP IDP config summary: %w", err)
	}
	return rows, nil
}
