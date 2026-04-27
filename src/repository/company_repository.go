package repository

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// CompanyRepository 회사 정보 레포지토리
type CompanyRepository struct {
	db *gorm.DB
}

// NewCompanyRepository 새 CompanyRepository 인스턴스 생성
func NewCompanyRepository(db *gorm.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// ExistsByRealmName realm_name으로 회사 존재 여부 확인
func (r *CompanyRepository) ExistsByRealmName(realmName string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Company{}).Where("realm_name = ?", realmName).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check company existence by realm_name: %w", err)
	}
	return count > 0, nil
}

// Create 회사 생성
func (r *CompanyRepository) Create(company *model.Company) error {
	if err := r.db.Create(company).Error; err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}
	return nil
}

// First 싱글톤 회사 조회 (첫 번째 레코드)
func (r *CompanyRepository) First() (*model.Company, error) {
	var company model.Company
	if err := r.db.First(&company).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	return &company, nil
}

// Save 회사 정보 저장 (업데이트)
func (r *CompanyRepository) Save(company *model.Company) error {
	if err := r.db.Save(company).Error; err != nil {
		return fmt.Errorf("failed to save company: %w", err)
	}
	return nil
}

// Count 회사 레코드 수 조회
func (r *CompanyRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&model.Company{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count companies: %w", err)
	}
	return count, nil
}
