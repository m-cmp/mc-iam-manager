package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/m-cmp/mc-iam-manager/model"
)

type CspRoleRepository struct {
	iamClient *iam.Client
}

func NewCspRoleRepository() (*CspRoleRepository, error) {
	// AWS SDK 설정
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	// IAM 클라이언트 생성
	client := iam.NewFromConfig(cfg)

	return &CspRoleRepository{
		iamClient: client,
	}, nil
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
				ID:          *role.RoleName,
				CspType:     "aws",
				CspRoleArn:  *role.Arn,
				Description: getRoleDescription(role),
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
func (r *CspRoleRepository) FindByCspType(cspType string) ([]*model.CspRole, error) {
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
			if role.RoleName == nil {
				continue
			}

			roleName := *role.RoleName
			//log.Printf("Checking role: %s", roleName)

			if len(roleName) >= 5 && strings.HasPrefix(roleName, "mciam") {
				roles = append(roles, &model.CspRole{
					Name:        roleName,
					CspType:     cspType,
					CspRoleArn:  *role.Arn,
					Description: getRoleDescription(role),
				})
				log.Printf("Added role: [Name: %s, ARN: %s, Description: %s, CreateDate: %v, Path: %s]",
					roleName,
					*role.Arn,
					getRoleDescription(role),
					role.CreateDate,
					*role.Path)
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

// getManagerAssumeRolePolicyDocument 플랫폼 관리자용 AssumeRole 정책 문서를 반환합니다.
func getRoleManagerAssumeRolePolicyDocument() string {
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

// Create AWS IAM Role을 생성합니다. : 플랫폼 관리자가 실행
func (r *CspRoleRepository) Create(role *model.CspRole) error {

	policyDocument := getRoleManagerAssumeRolePolicyDocument()

	// AWS IAM Role 생성
	input := &iam.CreateRoleInput{
		RoleName:                 &role.Name,
		AssumeRolePolicyDocument: aws.String(policyDocument),
		Description:              &role.Description,
	}

	result, err := r.iamClient.CreateRole(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to create IAM role: %v", err)
	}
	log.Printf("CreateRole API Response: [Role: {RoleName: %s, RoleId: %s, Arn: %s, CreateDate: %v, Path: %s, Description: %s, MaxSessionDuration: %d, PermissionsBoundary: %v, Tags: %v}]",
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

// Update AWS IAM Role을 수정합니다.
func (r *CspRoleRepository) Update(role *model.CspRole) error {
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
func (r *CspRoleRepository) Delete(id string) error {
	// AWS IAM Role 삭제
	input := &iam.DeleteRoleInput{
		RoleName: &id,
	}

	_, err := r.iamClient.DeleteRole(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to delete IAM role: %v", err)
	}
	log.Printf("DeleteRole API Response: [RoleName: %s] - Successfully deleted", id)
	return nil
}

func getRoleDescription(role types.Role) string {
	if role.Description != nil {
		return *role.Description
	}
	return ""
}
