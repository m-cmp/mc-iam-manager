package repository

import (
	"fmt"
	"time"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// TempCredentialRepository 임시 자격 증명 레포지토리
type TempCredentialRepository struct {
	db *gorm.DB
}

// NewTempCredentialRepository 새 TempCredentialRepository 인스턴스 생성
func NewTempCredentialRepository(db *gorm.DB) *TempCredentialRepository {
	return &TempCredentialRepository{
		db: db,
	}
}

// GetValidCredential 유효한 임시 자격 증명 조회 (만료되지 않은 것만)
func (r *TempCredentialRepository) GetValidCredential(provider, authType, region string, roleMasterID *uint, issuedBy string) (*model.TempCredential, error) {
	var credential model.TempCredential

	query := r.db.Where("provider = ? AND auth_type = ? AND region = ? AND is_active = ? AND expires_at > ? AND issued_by = ?",
		provider, authType, region, true, time.Now(), issuedBy)

	// RoleMaster ID가 제공된 경우 해당 ID로 필터링
	if roleMasterID != nil {
		query = query.Where("role_master_id = ?", *roleMasterID)
	} else {
		// RoleMaster ID가 없는 경우 NULL인 자격 증명만 조회
		query = query.Where("role_master_id IS NULL")
	}

	err := query.Order("expires_at DESC").First(&credential).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			roleMasterIDStr := "NULL"
			if roleMasterID != nil {
				roleMasterIDStr = fmt.Sprintf("%d", *roleMasterID)
			}
			return nil, fmt.Errorf("no valid credential found for provider: %s, authType: %s, region: %s, roleMasterID: %s, issuedBy: %s",
				provider, authType, region, roleMasterIDStr, issuedBy)
		}
		return nil, fmt.Errorf("failed to get valid credential: %w", err)
	}

	return &credential, nil
}

// CreateCredential 새로운 임시 자격 증명 생성
func (r *TempCredentialRepository) CreateCredential(credential *model.TempCredential) error {
	// 기존 자격 증명을 비활성화 (같은 provider, authType, region, roleMasterID)
	query := r.db.Model(&model.TempCredential{}).
		Where("provider = ? AND auth_type = ? AND region = ? AND is_active = ?",
			credential.Provider, credential.AuthType, credential.Region, true)

	if credential.RoleMasterID != nil {
		query = query.Where("role_master_id = ?", *credential.RoleMasterID)
	} else {
		query = query.Where("role_master_id IS NULL")
	}

	err := query.Update("is_active", false).Error
	if err != nil {
		return fmt.Errorf("failed to deactivate existing credentials: %w", err)
	}

	// 새로운 자격 증명 생성
	if err := r.db.Create(credential).Error; err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	return nil
}

// UpdateCredential 임시 자격 증명 업데이트
func (r *TempCredentialRepository) UpdateCredential(credential *model.TempCredential) error {
	if err := r.db.Save(credential).Error; err != nil {
		return fmt.Errorf("failed to update credential: %w", err)
	}
	return nil
}

// DeleteExpiredCredentials 만료된 자격 증명 삭제
func (r *TempCredentialRepository) DeleteExpiredCredentials() error {
	result := r.db.Where("expires_at <= ?", time.Now()).Delete(&model.TempCredential{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete expired credentials: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		fmt.Printf("Deleted %d expired credentials", result.RowsAffected)
	}

	return nil
}

// GetCredentialByID ID로 자격 증명 조회
func (r *TempCredentialRepository) GetCredentialByID(id uint) (*model.TempCredential, error) {
	var credential model.TempCredential
	if err := r.db.First(&credential, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get credential by ID: %w", err)
	}
	return &credential, nil
}

// ListCredentials 자격 증명 목록 조회 (필터링 가능)
func (r *TempCredentialRepository) ListCredentials(provider, authType string, roleMasterID *uint) ([]*model.TempCredential, error) {
	var credentials []*model.TempCredential

	query := r.db.Model(&model.TempCredential{})

	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if authType != "" {
		query = query.Where("auth_type = ?", authType)
	}
	if roleMasterID != nil {
		query = query.Where("role_master_id = ?", *roleMasterID)
	}

	if err := query.Order("created_at DESC").Find(&credentials).Error; err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	return credentials, nil
}

// DeactivateCredential 자격 증명 비활성화
func (r *TempCredentialRepository) DeactivateCredential(id uint) error {
	if err := r.db.Model(&model.TempCredential{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate credential: %w", err)
	}
	return nil
}

// GetOrCreateValidCredential 유효한 자격 증명이 있으면 반환, 없으면 생성 (팩토리 패턴)
func (r *TempCredentialRepository) GetOrCreateValidCredential(provider, authType, region string, roleMasterID *uint, issuedBy string,
	createFunc func() (*model.TempCredential, error)) (*model.TempCredential, error) {

	// 1. 유효한 자격 증명 조회 시도
	credential, err := r.GetValidCredential(provider, authType, region, roleMasterID, issuedBy)
	if err == nil && credential.IsValid() {
		return credential, nil
	}

	// 2. 유효한 자격 증명이 없으면 새로 생성
	newCredential, err := createFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create new credential: %w", err)
	}

	// RoleMaster ID와 IssuedBy 설정
	newCredential.RoleMasterID = roleMasterID
	newCredential.IssuedBy = issuedBy

	// 3. DB에 저장
	if err := r.CreateCredential(newCredential); err != nil {
		return nil, fmt.Errorf("failed to save new credential: %w", err)
	}

	return newCredential, nil
}

// GetCredentialsByRoleMasterID 특정 RoleMaster ID의 자격 증명 목록 조회
func (r *TempCredentialRepository) GetCredentialsByRoleMasterID(roleMasterID uint) ([]*model.TempCredential, error) {
	var credentials []*model.TempCredential

	if err := r.db.Where("role_master_id = ?", roleMasterID).Order("created_at DESC").Find(&credentials).Error; err != nil {
		return nil, fmt.Errorf("failed to get credentials by role master ID: %w", err)
	}

	return credentials, nil
}

// GetCredentialsByIssuedBy 특정 사용자가 발급한 자격 증명 목록 조회
func (r *TempCredentialRepository) GetCredentialsByIssuedBy(issuedBy string) ([]*model.TempCredential, error) {
	var credentials []*model.TempCredential

	if err := r.db.Where("issued_by = ?", issuedBy).Order("created_at DESC").Find(&credentials).Error; err != nil {
		return nil, fmt.Errorf("failed to get credentials by issued by: %w", err)
	}

	return credentials, nil
}
