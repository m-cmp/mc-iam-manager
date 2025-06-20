package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/google/uuid"
	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// CspRoleRepository CSP 역할 레포지토리
type CspRoleRepository struct {
	db        *gorm.DB
	iamClient *iam.Client
}

// NewCspRoleRepository 새 CspRoleRepository 인스턴스 생성
func NewCspRoleRepository(db *gorm.DB) *CspRoleRepository {
	// AWS SDK 설정
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil
	}

	// IAM 클라이언트 생성
	client := iam.NewFromConfig(cfg)

	return &CspRoleRepository{
		iamClient: client,
		db:        db,
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

		result, err := r.iamClient.ListRoles(context.TODO(), input)
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

		result, err := r.iamClient.ListRoles(context.TODO(), input)
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

type PolicyValues struct {
	AccountID        string
	KeycloakHostname string
	Subject          string
	Audience         string
}

// getManagerAssumeRolePolicyDocument 플랫폼 관리자용 AssumeRole 정책 문서를 반환합니다.
func getRoleManagerAssumeRolePolicyDocument(role *model.CspRole) (string, error) {
	const policyTemplate = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::{{.AccountID}}:oidc-provider/{{.KeycloakHostname}}"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {					
						"{{.KeycloakHostname}}:aud": "{{.Audience}}"
					}
				}
			}
		]
	}`
	// "{{.KeycloakHostname}}:sub": "{{.Subject}}",

	values := PolicyValues{
		AccountID:        "050864702683",
		KeycloakHostname: "mciam.onecloudcon.com",
		//Subject:          "user@example.com",
		Audience: "mciam-oidc-Client",
	}

	tmpl, err := template.New("policy").Parse(policyTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil

	// return `{
	// 	"Version": "2012-10-17",
	// 	"Statement": [
	// 		{
	// 			"Effect": "Allow",
	// 			"Principal": {
	// 				"Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME"
	// 			},
	// 			"Action": "sts:AssumeRoleWithWebIdentity",
	// 			"Condition": {
	// 				"StringEquals": {
	// 					"KEYCLOAK_HOSTNAME:sub": "SUBJECT",
	// 					"KEYCLOAK_HOSTNAME:aud": "AUDIENCE"
	// 				}
	// 			}
	// 		}
	// 	]
	// }`
}

// getUserAssumeRolePolicyDocument 일반 사용자용 AssumeRole 정책 문서를 반환합니다.
func getUserAssumeRolePolicyDocument() string {
	return `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"KEYCLOAK_HOSTNAME:sub": "SUBJECT",
						"KEYCLOAK_HOSTNAME:aud": "AUDIENCE"
					}
				}
			}
		]
	}`
}

// CreateCSPRole AWS IAM Role을 생성하고 생성 완료를 기다린 후 상세 정보를 반환합니다.
// 저장 전에 채워지는 Data가 있고 저장 후 채워지는 Data가 있음.
func (r *CspRoleRepository) CreateCSPRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	idpIdentifier := ""
	// 1. DB에서 역할 존재 여부 확인
	var existingRole model.CspRole

	// masterRoleId와 cspType을 이용하여 역할 존재 여부 확인
	newRole := &model.CspRole{
		Name:    req.RoleName,
		CspType: req.CspType,
	}
	if err := r.db.Where("name = ? AND csp_type = ?", req.RoleName, req.CspType).First(&existingRole).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("이미 존재합니다. 중복 생성 불가: %v", err)
		}
		log.Printf("db existing role: %v", existingRole)

		// DB에 역할이 없는 경우
		// 2. CSP에서 역할 존재 여부 확인
		getRoleInput := &iam.GetRoleInput{
			RoleName: aws.String(newRole.Name),
		}
		var targetCspRole *iam.GetRoleOutput
		if cspRoleResponse, err := r.iamClient.GetRole(context.TODO(), getRoleInput); err == nil {
			// CSP에 역할이 있으면 status를 created로 설정하여 저장
			newRole.Status = "created"
			targetCspRole = cspRoleResponse
		} else {
			// CSP에 역할이 없으면 status를 creating으로 설정하여 저장
			newRole.Status = "creating"
		}

		// 3. DB에 저장
		if err := r.db.Create(newRole).Error; err != nil {
			return nil, fmt.Errorf("failed to create CSP role in database: %v", err)
		}

		// 4. CSP에 역할이 없는 경우에만 생성
		if newRole.Status == "creating" {
			// AWS IAM Role 생성 요청
			assumeRolePolicyDocument, err := getRoleManagerAssumeRolePolicyDocument(newRole)
			if err != nil {
				newRole.Status = "failed"
				r.db.Save(newRole)
				return nil, fmt.Errorf("failed to generate assume role policy document: %v", err)
			}

			// assumeRolePolicyDocument에서 Principal.Federated 값 추출
			var policyDoc map[string]interface{}
			if err := json.Unmarshal([]byte(assumeRolePolicyDocument), &policyDoc); err != nil {
				newRole.Status = "failed"
				r.db.Save(newRole)
				return nil, fmt.Errorf("failed to parse assume role policy document: %v", err)
			}

			// Principal.Federated 값 추출하여 idpIdentifier에 설정
			if statements, ok := policyDoc["Statement"].([]interface{}); ok && len(statements) > 0 {
				if statement, ok := statements[0].(map[string]interface{}); ok {
					if principal, ok := statement["Principal"].(map[string]interface{}); ok {
						if federated, ok := principal["Federated"].(string); ok {
							idpIdentifier = federated
						}
					}
				}
			}

			input := &iam.CreateRoleInput{
				RoleName:                 aws.String(newRole.Name),
				AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
				Description:              aws.String(newRole.Description),
			}

			_, err = r.iamClient.CreateRole(context.TODO(), input)
			if err != nil {
				newRole.Status = "failed"
				r.db.Save(newRole)
				return nil, fmt.Errorf("failed to create IAM role: %v", err)
			}

			// 5. AWS IAM Role이 실제로 생성될 때까지 최대 5번, 1초 간격으로 확인
			for i := 0; i < 5; i++ {
				time.Sleep(1 * time.Second)

				getRoleResult, err := r.iamClient.GetRole(context.TODO(), getRoleInput)
				if err == nil && getRoleResult != nil && getRoleResult.Role != nil {
					// 역할이 생성되었으면 상세 정보 추출
					createdRole := &model.CspRole{
						ID:            newRole.ID,
						Name:          *getRoleResult.Role.RoleName,
						Description:   *getRoleResult.Role.Description,
						CspType:       newRole.CspType,
						IdpIdentifier: idpIdentifier,
						Status:        "created",
						CreateDate:    *getRoleResult.Role.CreateDate,
						Path:          *getRoleResult.Role.Path,
						IamRoleId:     *getRoleResult.Role.RoleId,
						IamIdentifier: *getRoleResult.Role.Arn,
					}

					// MaxSessionDuration 설정
					if getRoleResult.Role.MaxSessionDuration != nil {
						createdRole.MaxSessionDuration = getRoleResult.Role.MaxSessionDuration
					}

					// PermissionsBoundary 설정
					if getRoleResult.Role.PermissionsBoundary != nil && getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn != nil {
						createdRole.PermissionsBoundary = *getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn
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
						createdRole.RoleLastUsed = roleLastUsed
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
						createdRole.Tags = tags
					}

					// DB 업데이트
					if err := r.db.Save(createdRole).Error; err != nil {
						return nil, fmt.Errorf("failed to update CSP role in database: %v", err)
					}

					return createdRole, nil
				}
			}

			// 최대 시도 횟수를 초과하면 실패로 표시하고 DB 업데이트
			newRole.Status = "failed"
			if err := r.db.Save(newRole).Error; err != nil {
				return nil, fmt.Errorf("failed to update CSP role status to failed: %v", err)
			}
			return nil, fmt.Errorf("failed to verify IAM role creation after 5 attempts")
		} else {
			// 대상 csp에 role 정보를 db에 갱신
			newRole.Status = "created"
			newRole.IamIdentifier = *targetCspRole.Role.Arn
			newRole.CreateDate = *targetCspRole.Role.CreateDate
			newRole.Path = *targetCspRole.Role.Path
			newRole.IamRoleId = *targetCspRole.Role.RoleId

			if err := r.db.Save(newRole).Error; err != nil {
				return nil, fmt.Errorf("failed to update CSP role status to created: %v", err)
			}
		}

		return newRole, nil
	}

	// DB에 역할이 있는 경우
	// 1. status가 created인 경우 CSP에서 역할 존재 여부 확인
	if existingRole.Status == "created" {
		getRoleInput := &iam.GetRoleInput{
			RoleName: aws.String(newRole.Name),
		}
		if _, err := r.iamClient.GetRole(context.TODO(), getRoleInput); err == nil {
			return nil, fmt.Errorf("role already exists with status 'created'")
		}
	}

	// 2. status가 created가 아닌 경우
	// CSP에서 역할 존재 여부 확인
	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(newRole.Name),
	}
	if getRoleResult, err := r.iamClient.GetRole(context.TODO(), getRoleInput); err == nil && getRoleResult != nil && getRoleResult.Role != nil {
		log.Printf("csp existing role: %v", getRoleResult)
		// CSP에 역할이 있으면 status를 created로 업데이트
		existingRole.Status = "created"
		existingRole.IamIdentifier = *getRoleResult.Role.Arn
		existingRole.CreateDate = *getRoleResult.Role.CreateDate
		existingRole.Path = *getRoleResult.Role.Path
		existingRole.IamRoleId = *getRoleResult.Role.RoleId

		if getRoleResult.Role.MaxSessionDuration != nil {
			existingRole.MaxSessionDuration = getRoleResult.Role.MaxSessionDuration
		}

		if err := r.db.Save(&existingRole).Error; err != nil {
			return nil, fmt.Errorf("failed to update role status to created: %v", err)
		}
		return &existingRole, nil
	}

	// CSP에 역할이 없으면 생성
	assumeRolePolicyDocument, err := getRoleManagerAssumeRolePolicyDocument(newRole)
	if err != nil {
		existingRole.Status = "failed"
		r.db.Save(&existingRole)
		return nil, fmt.Errorf("failed to generate assume role policy document: %v", err)
	}

	// assumeRolePolicyDocument에서 Principal.Federated 값 추출
	var policyDoc map[string]interface{}
	if err := json.Unmarshal([]byte(assumeRolePolicyDocument), &policyDoc); err != nil {
		existingRole.Status = "failed"
		r.db.Save(&existingRole)
		return nil, fmt.Errorf("failed to parse assume role policy document: %v", err)
	}

	// Principal.Federated 값 추출하여 existingRole.IdpIdentifier에 설정
	if statements, ok := policyDoc["Statement"].([]interface{}); ok && len(statements) > 0 {
		if statement, ok := statements[0].(map[string]interface{}); ok {
			if principal, ok := statement["Principal"].(map[string]interface{}); ok {
				if federated, ok := principal["Federated"].(string); ok {
					existingRole.IdpIdentifier = federated
					// DB에 즉시 저장하여 IdpIdentifier 값이 유실되지 않도록 함
					if err := r.db.Save(&existingRole).Error; err != nil {
						return nil, fmt.Errorf("failed to save IdpIdentifier: %v", err)
					}
				}
			}
		}
	}

	input := &iam.CreateRoleInput{
		RoleName:                 aws.String(newRole.Name),
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
		Description:              aws.String(newRole.Description),
	}

	_, err = r.iamClient.CreateRole(context.TODO(), input)
	if err != nil {
		existingRole.Status = "failed"
		r.db.Save(&existingRole)
		return nil, fmt.Errorf("failed to create IAM role: %v", err)
	}

	// AWS IAM Role이 실제로 생성될 때까지 최대 5번, 1초 간격으로 확인
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)

		getRoleResult, err := r.iamClient.GetRole(context.TODO(), getRoleInput)
		if err == nil && getRoleResult != nil && getRoleResult.Role != nil {
			// 역할이 생성되었으면 상세 정보 추출
			existingRole.Status = "created"
			existingRole.IamIdentifier = *getRoleResult.Role.Arn
			existingRole.CreateDate = *getRoleResult.Role.CreateDate
			existingRole.Path = *getRoleResult.Role.Path
			existingRole.IamRoleId = *getRoleResult.Role.RoleId

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
				return nil, fmt.Errorf("failed to update CSP role in database: %v", err)
			}

			return &existingRole, nil
		}
	}

	// 최대 시도 횟수를 초과하면 실패로 표시하고 DB 업데이트
	existingRole.Status = "failed"
	if err := r.db.Save(&existingRole).Error; err != nil {
		return nil, fmt.Errorf("failed to update CSP role status to failed: %v", err)
	}
	return nil, fmt.Errorf("failed to verify IAM role creation after 5 attempts")
}

// Update AWS IAM Role을 수정합니다.
func (r *CspRoleRepository) UpdateCSPRole(role *model.CspRole) error {
	// AWS IAM Role 설명 업데이트
	input := &iam.UpdateRoleDescriptionInput{
		RoleName:    &role.Name,
		Description: &role.Description,
	}

	result, err := r.iamClient.UpdateRoleDescription(context.TODO(), input)
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

	_, err := r.iamClient.DeleteRole(context.TODO(), input)
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
func (r *CspRoleRepository) GetRoleByName(roleName string) (*model.CspRole, error) {
	var role model.CspRole
	if err := r.db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

// ExistCspRoleByName 이름으로 CSP 역할 존재 여부 확인 (CspRole 테이블에서)
func (r *CspRoleRepository) ExistCspRoleByName(roleName string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.CspRole{}).Where("name = ?", roleName).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check CSP role existence: %w", err)
	}
	return count > 0, nil
}

// ListAttachedRolePolicies 역할에 연결된 관리형 정책 목록 조회
func (r *CspRoleRepository) ListAttachedRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	result, err := r.iamClient.ListAttachedRolePolicies(ctx, input)
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
	input := &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	result, err := r.iamClient.ListRolePolicies(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list role policies: %w", err)
	}

	return result.PolicyNames, nil
}

// GetRolePolicy 역할의 특정 인라인 정책 조회
func (r *CspRoleRepository) GetRolePolicy(ctx context.Context, roleName string, policyName string) (*csp.RolePolicy, error) {
	input := &iam.GetRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	result, err := r.iamClient.GetRolePolicy(ctx, input)
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

	input := &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(string(policyDocument)),
	}

	_, err = r.iamClient.PutRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put role policy: %w", err)
	}

	return nil
}

// DeleteRolePolicy 역할에서 인라인 정책 삭제
func (r *CspRoleRepository) DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error {
	input := &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	_, err := r.iamClient.DeleteRolePolicy(ctx, input)
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
	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	getRoleResult, err := r.iamClient.GetRole(context.TODO(), getRoleInput)
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
