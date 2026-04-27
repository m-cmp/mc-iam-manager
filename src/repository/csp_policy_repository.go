package repository

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// CspPolicyRepository CSP 정책 레포지토리
type CspPolicyRepository struct {
	db *gorm.DB
}

// NewCspPolicyRepository 새 CspPolicyRepository 인스턴스 생성
func NewCspPolicyRepository(db *gorm.DB) *CspPolicyRepository {
	return &CspPolicyRepository{db: db}
}

// Create CSP 정책 생성
func (r *CspPolicyRepository) Create(policy *model.CspPolicy) error {
	if err := r.db.Create(policy).Error; err != nil {
		return fmt.Errorf("failed to create CSP policy: %w", err)
	}
	return nil
}

// GetByID ID로 CSP 정책 조회
func (r *CspPolicyRepository) GetByID(id uint) (*model.CspPolicy, error) {
	var policy model.CspPolicy
	if err := r.db.Preload("CspAccount").First(&policy, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP policy by ID: %w", err)
	}
	return &policy, nil
}

// GetByName 이름으로 CSP 정책 조회
func (r *CspPolicyRepository) GetByName(name string) (*model.CspPolicy, error) {
	var policy model.CspPolicy
	if err := r.db.Preload("CspAccount").Where("name = ?", name).First(&policy).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP policy by name: %w", err)
	}
	return &policy, nil
}

// GetByArn ARN으로 CSP 정책 조회
func (r *CspPolicyRepository) GetByArn(arn string) (*model.CspPolicy, error) {
	var policy model.CspPolicy
	if err := r.db.Preload("CspAccount").Where("policy_arn = ?", arn).First(&policy).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP policy by ARN: %w", err)
	}
	return &policy, nil
}

// GetByNameAndAccountID 이름과 계정 ID로 CSP 정책 조회
func (r *CspPolicyRepository) GetByNameAndAccountID(name string, accountID uint) (*model.CspPolicy, error) {
	var policy model.CspPolicy
	if err := r.db.Preload("CspAccount").Where("name = ? AND csp_account_id = ?", name, accountID).First(&policy).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP policy: %w", err)
	}
	return &policy, nil
}

// List CSP 정책 목록 조회
func (r *CspPolicyRepository) List(filter *model.CspPolicyFilter) ([]*model.CspPolicy, error) {
	var policies []*model.CspPolicy
	query := r.db.Model(&model.CspPolicy{}).Preload("CspAccount")

	if filter != nil {
		if filter.CspAccountID != nil {
			query = query.Where("csp_account_id = ?", *filter.CspAccountID)
		}
		if filter.PolicyType != "" {
			query = query.Where("policy_type = ?", filter.PolicyType)
		}
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
	}

	if err := query.Order("created_at DESC").Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to list CSP policies: %w", err)
	}
	return policies, nil
}

// Update CSP 정책 수정
func (r *CspPolicyRepository) Update(policy *model.CspPolicy) error {
	if err := r.db.Save(policy).Error; err != nil {
		return fmt.Errorf("failed to update CSP policy: %w", err)
	}
	return nil
}

// Delete CSP 정책 삭제
func (r *CspPolicyRepository) Delete(id uint) error {
	// 트랜잭션으로 매핑 테이블도 함께 삭제
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 매핑 관계 삭제
		if err := tx.Where("csp_policy_id = ?", id).Delete(&model.CspRolePolicyMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete policy mappings: %w", err)
		}

		// 정책 삭제
		result := tx.Delete(&model.CspPolicy{}, id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete CSP policy: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("CSP policy not found")
		}
		return nil
	})
}

// ExistsByID ID로 CSP 정책 존재 여부 확인
func (r *CspPolicyRepository) ExistsByID(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspPolicy{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP policy existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByName 이름으로 CSP 정책 존재 여부 확인
func (r *CspPolicyRepository) ExistsByName(name string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspPolicy{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP policy existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByArn ARN으로 CSP 정책 존재 여부 확인
func (r *CspPolicyRepository) ExistsByArn(arn string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspPolicy{}).Where("policy_arn = ?", arn).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP policy existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameAndAccountID 이름과 계정 ID로 CSP 정책 존재 여부 확인
func (r *CspPolicyRepository) ExistsByNameAndAccountID(name string, accountID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspPolicy{}).Where("name = ? AND csp_account_id = ?", name, accountID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP policy existence: %w", err)
	}
	return count > 0, nil
}

// GetByAccountID 계정 ID로 CSP 정책 목록 조회
func (r *CspPolicyRepository) GetByAccountID(accountID uint) ([]*model.CspPolicy, error) {
	var policies []*model.CspPolicy
	if err := r.db.Preload("CspAccount").Where("csp_account_id = ?", accountID).Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP policies by account ID: %w", err)
	}
	return policies, nil
}

// GetByPolicyType 정책 타입으로 CSP 정책 목록 조회
func (r *CspPolicyRepository) GetByPolicyType(policyType model.PolicyType) ([]*model.CspPolicy, error) {
	var policies []*model.CspPolicy
	if err := r.db.Preload("CspAccount").Where("policy_type = ?", policyType).Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP policies by type: %w", err)
	}
	return policies, nil
}

// AttachPolicyToRole 역할에 정책 연결
func (r *CspPolicyRepository) AttachPolicyToRole(roleID, policyID uint) error {
	mapping := model.CspRolePolicyMapping{
		CspRoleID:   roleID,
		CspPolicyID: policyID,
	}
	if err := r.db.Create(&mapping).Error; err != nil {
		return fmt.Errorf("failed to attach policy to role: %w", err)
	}
	return nil
}

// DetachPolicyFromRole 역할에서 정책 분리
func (r *CspPolicyRepository) DetachPolicyFromRole(roleID, policyID uint) error {
	result := r.db.Where("csp_role_id = ? AND csp_policy_id = ?", roleID, policyID).Delete(&model.CspRolePolicyMapping{})
	if result.Error != nil {
		return fmt.Errorf("failed to detach policy from role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("policy mapping not found")
	}
	return nil
}

// GetPoliciesByRoleID 역할에 연결된 정책 목록 조회
func (r *CspPolicyRepository) GetPoliciesByRoleID(roleID uint) ([]*model.CspPolicy, error) {
	var policies []*model.CspPolicy
	err := r.db.
		Joins("JOIN mcmp_csp_role_policy_mappings ON mcmp_csp_role_policy_mappings.csp_policy_id = mcmp_csp_policies.id").
		Where("mcmp_csp_role_policy_mappings.csp_role_id = ?", roleID).
		Preload("CspAccount").
		Find(&policies).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by role ID: %w", err)
	}
	return policies, nil
}

// GetRolesByPolicyID 정책이 연결된 역할 목록 조회
func (r *CspPolicyRepository) GetRolesByPolicyID(policyID uint) ([]*model.CspRole, error) {
	var roles []*model.CspRole
	err := r.db.
		Joins("JOIN mcmp_csp_role_policy_mappings ON mcmp_csp_role_policy_mappings.csp_role_id = mcmp_role_csp_roles.id").
		Where("mcmp_csp_role_policy_mappings.csp_policy_id = ?", policyID).
		Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get roles by policy ID: %w", err)
	}
	return roles, nil
}

// IsPolicyAttachedToRole 역할에 정책이 연결되어 있는지 확인
func (r *CspPolicyRepository) IsPolicyAttachedToRole(roleID, policyID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspRolePolicyMapping{}).
		Where("csp_role_id = ? AND csp_policy_id = ?", roleID, policyID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check policy attachment: %w", err)
	}
	return count > 0, nil
}

// CountByAccountID 특정 계정의 CSP 정책 개수 조회
func (r *CspPolicyRepository) CountByAccountID(accountID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.CspPolicy{}).Where("csp_account_id = ?", accountID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count CSP policies: %w", err)
	}
	return count, nil
}

// GetManagedPoliciesByAccountID 특정 계정의 관리형 정책 목록 조회
func (r *CspPolicyRepository) GetManagedPoliciesByAccountID(accountID uint) ([]*model.CspPolicy, error) {
	var policies []*model.CspPolicy
	if err := r.db.Preload("CspAccount").
		Where("csp_account_id = ? AND policy_type = ?", accountID, model.PolicyTypeManaged).
		Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to get managed policies: %w", err)
	}
	return policies, nil
}
