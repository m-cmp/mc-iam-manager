package repository

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// CspAccountRepository CSP 계정 레포지토리
type CspAccountRepository struct {
	db *gorm.DB
}

// NewCspAccountRepository 새 CspAccountRepository 인스턴스 생성
func NewCspAccountRepository(db *gorm.DB) *CspAccountRepository {
	return &CspAccountRepository{db: db}
}

// Create CSP 계정 생성
func (r *CspAccountRepository) Create(account *model.CspAccount) error {
	if err := r.db.Create(account).Error; err != nil {
		return fmt.Errorf("failed to create CSP account: %w", err)
	}
	return nil
}

// GetByID ID로 CSP 계정 조회
func (r *CspAccountRepository) GetByID(id uint) (*model.CspAccount, error) {
	var account model.CspAccount
	if err := r.db.First(&account, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP account by ID: %w", err)
	}
	return &account, nil
}

// GetByName 이름으로 CSP 계정 조회
func (r *CspAccountRepository) GetByName(name string) (*model.CspAccount, error) {
	var account model.CspAccount
	if err := r.db.Where("name = ?", name).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP account by name: %w", err)
	}
	return &account, nil
}

// GetByNameAndCspType 이름과 CSP 타입으로 CSP 계정 조회
func (r *CspAccountRepository) GetByNameAndCspType(name string, cspType string) (*model.CspAccount, error) {
	var account model.CspAccount
	if err := r.db.Where("name = ? AND csp_type = ?", name, cspType).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	return &account, nil
}

// List CSP 계정 목록 조회
func (r *CspAccountRepository) List(filter *model.CspAccountFilter) ([]*model.CspAccount, error) {
	var accounts []*model.CspAccount
	query := r.db.Model(&model.CspAccount{})

	if filter != nil {
		if filter.CspType != "" {
			query = query.Where("csp_type = ?", filter.CspType)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
	}

	if err := query.Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to list CSP accounts: %w", err)
	}
	return accounts, nil
}

// Update CSP 계정 수정
func (r *CspAccountRepository) Update(account *model.CspAccount) error {
	if err := r.db.Save(account).Error; err != nil {
		return fmt.Errorf("failed to update CSP account: %w", err)
	}
	return nil
}

// Delete CSP 계정 삭제
func (r *CspAccountRepository) Delete(id uint) error {
	result := r.db.Delete(&model.CspAccount{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete CSP account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("CSP account not found")
	}
	return nil
}

// ExistsByID ID로 CSP 계정 존재 여부 확인
func (r *CspAccountRepository) ExistsByID(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspAccount{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP account existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByName 이름으로 CSP 계정 존재 여부 확인
func (r *CspAccountRepository) ExistsByName(name string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspAccount{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP account existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameAndCspType 이름과 CSP 타입으로 CSP 계정 존재 여부 확인
func (r *CspAccountRepository) ExistsByNameAndCspType(name string, cspType string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspAccount{}).Where("name = ? AND csp_type = ?", name, cspType).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP account existence: %w", err)
	}
	return count > 0, nil
}

// GetActiveAccounts 활성 CSP 계정 목록 조회
func (r *CspAccountRepository) GetActiveAccounts() ([]*model.CspAccount, error) {
	var accounts []*model.CspAccount
	if err := r.db.Where("is_active = ?", true).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get active CSP accounts: %w", err)
	}
	return accounts, nil
}

// GetByCspType CSP 타입으로 계정 목록 조회
func (r *CspAccountRepository) GetByCspType(cspType string) ([]*model.CspAccount, error) {
	var accounts []*model.CspAccount
	if err := r.db.Where("csp_type = ?", cspType).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP accounts by type: %w", err)
	}
	return accounts, nil
}
