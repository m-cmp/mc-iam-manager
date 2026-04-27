package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/google/uuid"
	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// CspRoleRepository CSP 역할 레포지토리
type CspRoleRepository struct {
	db                 *gorm.DB
	tempCredentialRepo *TempCredentialRepository
}

// NewCspRoleRepository 새 CspRoleRepository 인스턴스 생성
func NewCspRoleRepository(db *gorm.DB) *CspRoleRepository {
	return &CspRoleRepository{
		db:                 db,
		tempCredentialRepo: NewTempCredentialRepository(db),
	}
}

// FindAll AWS IAM Role 목록을 조회합니다.
func (r *CspRoleRepository) FindAll() ([]*model.CspRole, error) {
	var roles []*model.CspRole
	var marker *string

	for {
		// AWS IAM Role 목록 조회 (페이지네이션)
		input := &iam.ListRolesInput{
			Marker: marker,
		}

		awsIamClient, err := r.getAwsIamClient("system")
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS IAM client: %v", err)
		}

		result, err := awsIamClient.ListRoles(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM roles: %v", err)
		}

		// 현재 페이지의 역할들을 처리
		for _, role := range result.Roles {
			roles = append(roles, &model.CspRole{
				Name:          *role.RoleName,
				CspType:       "aws",
				IamIdentifier: *role.Arn,
				Description:   getRoleDescription(role),
			})
		}

		// 다음 페이지가 있는지 확인
		if !result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	log.Printf("Found %d All roles in AWS IAM", len(roles))
	return roles, nil
}

// FindByCspType IAM Role 목록을 조회합니다. AWS는 ListRoles에서 Tag, 각종 filter조건을 지원하지 않음
func (r *CspRoleRepository) FindMciamRoleFromCsp(cspType string) ([]*model.CspRole, error) {
	var roles []*model.CspRole
	var marker *string

	for {
		// AWS IAM Role 목록 조회 (페이지네이션)
		input := &iam.ListRolesInput{
			Marker: marker, // for pagenation
		}

		awsIamClient, err := r.getAwsIamClient("system")
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS IAM client: %v", err)
		}

		result, err := awsIamClient.ListRoles(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM roles: %v", err)
		}

		// 현재 페이지의 역할들을 처리
		for _, role := range result.Roles {
			if role.RoleName == nil {
				continue
			}

			roleName := *role.RoleName
			//log.Printf("Checking role: %s", roleName)

			if len(roleName) >= 5 && strings.HasPrefix(roleName, "mciam") {
				roles = append(roles, &model.CspRole{
					Name:          roleName,
					CspType:       cspType,
					IamIdentifier: *role.Arn,
					Description:   getRoleDescription(role),
				})
				// log.Printf("Added role: [Name: %s, ARN: %s, Description: %s, CreateDate: %v, Path: %s]",
				// 	roleName,
				// 	*role.Arn,
				// 	getRoleDescription(role),
				// 	role.CreateDate,
				// 	*role.Path)
				b, _ := json.MarshalIndent(role, "", "  ")
				log.Println(string(b))
			}

		}

		// 다음 페이지가 있는지 확인
		if !result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	log.Printf("Found %d mciam roles in %v", len(roles), cspType)
	return roles, nil
}

// CreateCspRoleRecord DB에 CSP 역할 레코드를 생성합니다. (DB 전용)
func (r *CspRoleRepository) CreateCspRoleRecord(role *model.CspRole) error {
	if err := r.db.Create(role).Error; err != nil {
		return fmt.Errorf("failed to create CSP role in database: %v", err)
	}
	return nil
}

// UpdateCspRoleRecord DB의 CSP 역할 레코드를 업데이트합니다. (DB 전용)
func (r *CspRoleRepository) UpdateCspRoleRecord(role *model.CspRole) error {
	if err := r.db.Save(role).Error; err != nil {
		return fmt.Errorf("failed to update CSP role in database: %v", err)
	}
	return nil
}

// createAwsConfigWithTempCredential 임시 자격 증명으로 AWS 설정을 생성합니다.
func (r *CspRoleRepository) createAwsConfigWithTempCredential(credential *model.TempCredential) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load default AWS config: %v", err)
	}
	cfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		credential.AccessKeyId,
		credential.SecretAccessKey,
		credential.SessionToken,
	))
	cfg.Region = credential.Region
	return cfg, nil
}

// Update AWS IAM Role을 수정합니다.
func (r *CspRoleRepository) UpdateCSPRole(role *model.CspRole) error {
	// AWS IAM Role 설명 업데이트
	input := &iam.UpdateRoleDescriptionInput{
		RoleName:    &role.Name,
		Description: &role.Description,
	}

	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	result, err := awsIamClient.UpdateRoleDescription(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to update IAM role: %v", err)
	}
	log.Printf("UpdateRoleDescription API Response: [Role: {RoleName: %s, RoleId: %s, Arn: %s, CreateDate: %v, Path: %s, Description: %s, MaxSessionDuration: %d, PermissionsBoundary: %v, Tags: %v}]",
		*result.Role.RoleName,
		*result.Role.RoleId,
		*result.Role.Arn,
		result.Role.CreateDate,
		*result.Role.Path,
		*result.Role.Description,
		*result.Role.MaxSessionDuration,
		result.Role.PermissionsBoundary,
		result.Role.Tags)
	return nil
}

// Delete AWS IAM Role을 삭제합니다.
func (r *CspRoleRepository) DeleteCSPRole(id string) error {
	// 1. DB에서 역할 조회 (role_master 테이블 조인)
	var role model.CspRole
	if err := r.db.Joins("JOIN mcmp_role_csp_role_mappings ON mcmp_role_csp_role_mappings.csp_role_id = mcmp_role_csps.id").
		Where("mcmp_role_csp_role_mappings.csp_role_id = ?", id).
		First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("CSP 역할을 찾을 수 없습니다: %v", err)
		}
		return fmt.Errorf("DB 조회 실패: %v", err)
	}

	// 2. AWS IAM Role 삭제
	input := &iam.DeleteRoleInput{
		RoleName: aws.String(role.Name),
	}

	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	_, err = awsIamClient.DeleteRole(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to delete IAM role: %v", err)
	}

	// 3. DB에서도 삭제
	if err := r.db.Delete(&role).Error; err != nil {
		return fmt.Errorf("failed to delete role from database: %v", err)
	}

	log.Printf("DeleteRole API Response: [RoleName: %s] - Successfully deleted", role.Name)
	return nil
}

func getRoleDescription(role types.Role) string {
	if role.Description != nil {
		return *role.Description
	}
	return ""
}

// AddPermissionsToCSPRole CSP 역할에 권한을 추가합니다.
func (r *CspRoleRepository) AddPermissionsToCSPRole(roleID string, permissions []string) error {
	// 트랜잭션 시작
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 기존 권한 조회
	var existingPermissions []model.CspRolePermission
	if err := tx.Where("csp_role_id = ?", roleID).Find(&existingPermissions).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 기존 권한 맵 생성
	existingPermMap := make(map[string]bool)
	for _, p := range existingPermissions {
		existingPermMap[p.Permission] = true
	}

	// 새로운 권한 추가
	for _, permission := range permissions {
		if !existingPermMap[permission] {
			newPermission := model.CspRolePermission{
				ID:         uuid.New().String(),
				CspRoleID:  roleID,
				Permission: permission,
			}
			if err := tx.Create(&newPermission).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// 트랜잭션 커밋
	return tx.Commit().Error
}

// RemovePermissionsFromCSPRole CSP 역할에서 권한을 제거합니다.
func (r *CspRoleRepository) RemovePermissionsFromCSPRole(roleID string, permissions []string) error {
	return r.db.Where("csp_role_id = ? AND permission IN ?", roleID, permissions).Delete(&model.CspRolePermission{}).Error
}

// GetCSPRolePermissions CSP 역할의 권한 목록을 조회합니다.
func (r *CspRoleRepository) GetCSPRolePermissions(roleID string) ([]string, error) {
	var permissions []model.CspRolePermission
	if err := r.db.Where("csp_role_id = ?", roleID).Find(&permissions).Error; err != nil {
		return nil, err
	}

	// 권한 문자열 목록으로 변환
	permissionStrings := make([]string, len(permissions))
	for i, p := range permissions {
		permissionStrings[i] = p.Permission
	}
	return permissionStrings, nil
}

// GetRole 역할 정보 조회
func (r *CspRoleRepository) GetRoleByID(cspRoleId uint) (*model.CspRole, error) {
	var role model.CspRole
	if err := r.db.Where("id = ?", cspRoleId).First(&role).Error; err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

// GetRole 역할 정보 조회
func (r *CspRoleRepository) GetCspRoleByName(roleName string, cspType string) (*model.CspRole, error) {
	var role model.CspRole
	if err := r.db.Where("name = ? AND csp_type = ?", roleName, cspType).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 역할이 존재하지 않음
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

// GetRole 역할 정보 조회. 같은이름의 역할에 cspType 이 다를 수 있음
func (r *CspRoleRepository) GetCspRolesByName(roleName string) ([]*model.CspRole, error) {
	var roles []*model.CspRole
	if err := r.db.Where("name = ?", roleName).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ExistCspRoleByName 이름으로 CSP 역할 존재 여부 확인 (CspRole 테이블에서)
func (r *CspRoleRepository) ExistCspRoleByName(roleName string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspRole{}).Where("name = ?", roleName).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP role existence: %w", err)
	}
	return count > 0, nil
}

func (r *CspRoleRepository) ExistCspRoleByNameAndType(roleName string, cspType string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspRole{}).Where("name = ? AND csp_type = ?", roleName, cspType).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP role existence: %w", err)
	}
	return count > 0, nil
}

// ListAttachedRolePolicies 역할에 연결된 관리형 정책 목록 조회
func (r *CspRoleRepository) ListAttachedRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	result, err := awsIamClient.ListAttachedRolePolicies(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list attached role policies: %w", err)
	}

	policies := make([]string, 0, len(result.AttachedPolicies))
	for _, policy := range result.AttachedPolicies {
		policies = append(policies, *policy.PolicyArn)
	}

	return policies, nil
}

// ListRolePolicies 역할의 인라인 정책 목록 조회
func (r *CspRoleRepository) ListRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	input := &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	result, err := awsIamClient.ListRolePolicies(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list role policies: %w", err)
	}

	return result.PolicyNames, nil
}

// GetRolePolicy 역할의 특정 인라인 정책 조회
func (r *CspRoleRepository) GetRolePolicy(ctx context.Context, roleName string, policyName string) (*csp.RolePolicy, error) {
	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	input := &iam.GetRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	result, err := awsIamClient.GetRolePolicy(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get role policy: %w", err)
	}

	var policy csp.RolePolicy
	if err := json.Unmarshal([]byte(*result.PolicyDocument), &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy document: %w", err)
	}

	return &policy, nil
}

// PutRolePolicy 역할에 인라인 정책 추가/수정
func (r *CspRoleRepository) PutRolePolicy(ctx context.Context, roleName string, policyName string, policy *csp.RolePolicy) error {
	policyDocument, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	input := &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(string(policyDocument)),
	}

	_, err = awsIamClient.PutRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put role policy: %w", err)
	}

	return nil
}

// DeleteRolePolicy 역할에서 인라인 정책 삭제
func (r *CspRoleRepository) DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error {
	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	input := &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	_, err = awsIamClient.DeleteRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete role policy: %w", err)
	}

	return nil
}

// ExistsCspRoleByID ID로 CSP 역할 존재 여부 확인
func (r *CspRoleRepository) ExistsCspRoleByID(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspRole{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// SyncCspRoleFromCloud 실제 CSP에서 역할 정보를 조회하여 DB를 업데이트합니다.
func (r *CspRoleRepository) SyncCspRoleFromCloud(roleName string) (*model.CspRole, error) {
	// AWS IAM에서 역할 정보 조회
	awsIamClient, err := r.getAwsIamClient("system")
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS IAM client: %v", err)
	}

	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	getRoleResult, err := awsIamClient.GetRole(context.TODO(), getRoleInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get role from AWS IAM: %w", err)
	}

	if getRoleResult == nil || getRoleResult.Role == nil {
		return nil, fmt.Errorf("role not found in AWS IAM: %s", roleName)
	}

	// DB에서 기존 역할 조회
	var existingRole model.CspRole
	if err := r.db.Where("name = ?", roleName).First(&existingRole).Error; err != nil {
		return nil, fmt.Errorf("failed to get existing role from database: %w", err)
	}

	// AWS IAM에서 가져온 정보로 업데이트
	existingRole.Status = "created"
	existingRole.IamIdentifier = *getRoleResult.Role.Arn
	existingRole.CreateDate = *getRoleResult.Role.CreateDate
	existingRole.Path = *getRoleResult.Role.Path
	existingRole.IamRoleId = *getRoleResult.Role.RoleId

	// Description 업데이트
	if getRoleResult.Role.Description != nil {
		existingRole.Description = *getRoleResult.Role.Description
	}

	// MaxSessionDuration 설정
	if getRoleResult.Role.MaxSessionDuration != nil {
		existingRole.MaxSessionDuration = getRoleResult.Role.MaxSessionDuration
	}

	// PermissionsBoundary 설정
	if getRoleResult.Role.PermissionsBoundary != nil && getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn != nil {
		existingRole.PermissionsBoundary = *getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn
	}

	// RoleLastUsed 설정
	if getRoleResult.Role.RoleLastUsed != nil {
		roleLastUsed := &model.RoleLastUsed{}
		if getRoleResult.Role.RoleLastUsed.LastUsedDate != nil {
			roleLastUsed.LastUsedDate = *getRoleResult.Role.RoleLastUsed.LastUsedDate
		}
		if getRoleResult.Role.RoleLastUsed.Region != nil {
			roleLastUsed.Region = *getRoleResult.Role.RoleLastUsed.Region
		}
		existingRole.RoleLastUsed = roleLastUsed
	}

	// Tags 설정
	if len(getRoleResult.Role.Tags) > 0 {
		tags := make([]model.Tag, len(getRoleResult.Role.Tags))
		for i, tag := range getRoleResult.Role.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[i] = model.Tag{
					Key:   *tag.Key,
					Value: *tag.Value,
				}
			}
		}
		existingRole.Tags = tags
	}

	// DB 업데이트
	if err := r.db.Save(&existingRole).Error; err != nil {
		return nil, fmt.Errorf("failed to update CSP role in database: %w", err)
	}

	return &existingRole, nil
}

// getAwsIamClient 임시 자격 증명으로 AWS IAM 클라이언트를 생성합니다.
func (r *CspRoleRepository) getAwsIamClient(issuedBy string) (*iam.Client, error) {
	// 유효한 임시 자격 증명 조회 (시스템 레벨)
	credential, err := r.tempCredentialRepo.GetValidCredential("aws", "oidc", "ap-northeast-2", nil, issuedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid credential: %v", err)
	}

	// AWS 설정 생성
	cfg, err := r.createAwsConfigWithTempCredential(credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %v", err)
	}

	return iam.NewFromConfig(cfg), nil
}

// GetByCspAccountID CspAccountID로 CspRole 목록 조회 (CspIdpConfig Preload)
func (r *CspRoleRepository) GetByCspAccountID(accountID uint) ([]*model.CspRole, error) {
	var roles []*model.CspRole
	if err := r.db.Preload("CspIdpConfig").Where("csp_account_id = ?", accountID).Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("failed to get CSP roles by account ID: %w", err)
	}
	return roles, nil
}
