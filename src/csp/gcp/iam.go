package gcp

import (
	"context"
	"fmt"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/m-cmp/mc-iam-manager/csp"
	"google.golang.org/api/option"
)

// GCPIAMClient GCP IAM 클라이언트
type GCPIAMClient struct {
	client *admin.IamClient
	config *csp.IAMClientConfig
}

// NewGCPIAMClient 새로운 GCP IAM 클라이언트 생성
func NewGCPIAMClient(cfg *csp.IAMClientConfig) (*GCPIAMClient, error) {
	ctx := context.Background()

	// GCP IAM 클라이언트 생성
	client, err := admin.NewIamClient(ctx, option.WithCredentialsFile(cfg.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	return &GCPIAMClient{
		client: client,
		config: cfg,
	}, nil
}

// CreateRole IAM 역할 생성
func (c *GCPIAMClient) CreateRole(ctx context.Context, role *csp.Role) error {
	// 정책 문서를 JSON 문자열로 변환
	// gcpPolicyDocument, err := json.Marshal(role.Policy)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal policy document: %w", err)
	// }

	// // 역할 생성 요청
	// req := &adminpb.CreateRoleRequest{
	// 	Parent: fmt.Sprintf("projects/%s", c.config.ProjectID),
	// 	RoleId: role.Name,
	// 	Role: &adminpb.Role{
	// 		Title:       role.Name,
	// 		Description: role.Description,
	// 		Permissions: convertPermissions(role.Policy),
	// 		Stage:       adminpb.Role_GA,
	// 	},
	// }

	// _, err = c.client.CreateRole(ctx, req)
	// if err != nil {
	// 	return fmt.Errorf("failed to create role: %w", err)
	// }

	return nil
}

// DeleteRole IAM 역할 삭제
func (c *GCPIAMClient) DeleteRole(ctx context.Context, roleName string) error {
	// req := &adminpb.DeleteRoleRequest{
	// 	Name: fmt.Sprintf("projects/%s/roles/%s", c.config.ProjectID, roleName),
	// }

	// err := c.client.DeleteRole(ctx, req)
	// if err != nil {
	// 	return fmt.Errorf("failed to delete role: %w", err)
	// }

	return nil
}

// GetRole IAM 역할 정보 조회
func (c *GCPIAMClient) GetRole(ctx context.Context, roleName string) (*csp.Role, error) {
	// req := &adminpb.GetRoleRequest{
	// 	//Name: fmt.Sprintf("projects/%s/roles/%s", c.config.ProjectID, roleName),
	// }

	// role, err := c.client.GetRole(ctx, req)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get role: %w", err)
	// }

	// // 정책 문서 생성
	// policy := &csp.RolePolicy{
	// 	Version: "2",
	// 	Statement: []csp.RolePolicyStatement{
	// 		{
	// 			Effect:   "Allow",
	// 			Action:   role.Permissions,
	// 			Resource: []string{"*"},
	// 		},
	// 	},
	// }

	// return &csp.Role{
	// 	Name:        role.Name,
	// 	Description: role.Description,
	// 	Policy:      policy,
	// 	Tags:        convertGCPTags(role),
	// }, nil
	return nil, nil
}

// UpdateRole IAM 역할 정보 수정
func (c *GCPIAMClient) UpdateRole(ctx context.Context, role *csp.Role) error {
	// req := &adminpb.RoleMasterSubRequest{
	// 	Name: fmt.Sprintf("projects/%s/roles/%s", c.config.ProjectID, role.Name),
	// 	Role: &adminpb.Role{
	// 		Title:       role.Name,
	// 		Description: role.Description,
	// 		Permissions: convertPermissions(role.Policy),
	// 	},
	// 	UpdateMask: &adminpb.FieldMask{
	// 		Paths: []string{"title", "description", "permissions"},
	// 	},
	// }

	// _, err := c.client.UpdateRole(ctx, req)
	// if err != nil {
	// 	return fmt.Errorf("failed to update role: %w", err)
	// }

	return nil
}

// AttachRolePolicy IAM 역할에 정책 연결
func (c *GCPIAMClient) AttachRolePolicy(ctx context.Context, roleName string, policyArn string) error {
	// // GCP에서는 정책을 직접 연결하는 대신 역할에 권한을 추가
	// role, err := c.GetRole(ctx, roleName)
	// if err != nil {
	// 	return err
	// }

	// // 정책 문서 파싱
	// var policy csp.RolePolicy
	// if err := json.Unmarshal([]byte(policyArn), &policy); err != nil {
	// 	return fmt.Errorf("failed to parse policy document: %w", err)
	// }

	// // 기존 권한에 새로운 권한 추가
	// role.Policy.Statement = append(role.Policy.Statement, policy.Statement...)

	// // 역할 업데이트
	// return c.UpdateRole(ctx, role)
	return nil
}

// DetachRolePolicy IAM 역할에서 정책 분리
func (c *GCPIAMClient) DetachRolePolicy(ctx context.Context, roleName string, policyArn string) error {
	// // GCP에서는 정책을 직접 분리하는 대신 역할에서 권한을 제거
	// role, err := c.GetRole(ctx, roleName)
	// if err != nil {
	// 	return err
	// }

	// // 정책 문서 파싱
	// var policy csp.RolePolicy
	// if err := json.Unmarshal([]byte(policyArn), &policy); err != nil {
	// 	return fmt.Errorf("failed to parse policy document: %w", err)
	// }

	// // 제거할 권한 목록 생성
	// removePermissions := make(map[string]bool)
	// for _, stmt := range policy.Statement {
	// 	for _, action := range stmt.Action {
	// 		removePermissions[action] = true
	// 	}
	// }

	// // 기존 권한에서 제거할 권한 필터링
	// var newStatements []csp.RolePolicyStatement
	// for _, stmt := range role.Policy.Statement {
	// 	var newActions []string
	// 	for _, action := range stmt.Action {
	// 		if !removePermissions[action] {
	// 			newActions = append(newActions, action)
	// 		}
	// 	}
	// 	if len(newActions) > 0 {
	// 		stmt.Action = newActions
	// 		newStatements = append(newStatements, stmt)
	// 	}
	// }

	// role.Policy.Statement = newStatements

	// // 역할 업데이트
	// return c.UpdateRole(ctx, role)
	return nil
}

// ListRolePolicies IAM 역할에 연결된 정책 목록 조회
func (c *GCPIAMClient) ListRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	// role, err := c.GetRole(ctx, roleName)
	// if err != nil {
	// 	return nil, err
	// }

	// // 권한을 정책 형식으로 변환
	// policies := make([]string, 0, len(role.Policy.Statement))
	// for _, stmt := range role.Policy.Statement {
	// 	policy := &csp.RolePolicy{
	// 		Version:   "2",
	// 		Statement: []csp.RolePolicyStatement{stmt},
	// 	}
	// 	policyJSON, err := json.Marshal(policy)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	// 	}
	// 	policies = append(policies, string(policyJSON))
	// }

	// return policies, nil
	return nil, nil
}

// GetRolePolicy IAM 역할의 특정 정책 조회
func (c *GCPIAMClient) GetRolePolicy(ctx context.Context, roleName string, policyName string) (*csp.RolePolicy, error) {
	// role, err := c.GetRole(ctx, roleName)
	// if err != nil {
	// 	return nil, err
	// }

	// // 정책 이름으로 해당 정책 찾기
	// for _, stmt := range role.Policy.Statement {
	// 	policy := &csp.RolePolicy{
	// 		Version:   "2",
	// 		Statement: []csp.RolePolicyStatement{stmt},
	// 	}
	// 	policyJSON, err := json.Marshal(policy)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	// 	}
	// 	if string(policyJSON) == policyName {
	// 		return policy, nil
	// 	}
	// }

	return nil, fmt.Errorf("policy not found: %s", policyName)
}

// PutRolePolicy IAM 역할에 정책 추가/수정
func (c *GCPIAMClient) PutRolePolicy(ctx context.Context, roleName string, policyName string, policy *csp.RolePolicy) error {
	// role, err := c.GetRole(ctx, roleName)
	// if err != nil {
	// 	return err
	// }

	// // 기존 정책 제거
	// if err := c.DetachRolePolicy(ctx, roleName, policyName); err != nil {
	// 	return err
	// }

	// // 새 정책 추가
	// return c.AttachRolePolicy(ctx, roleName, policyName)

	return nil
}

// DeleteRolePolicy IAM 역할에서 정책 삭제
func (c *GCPIAMClient) DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error {
	return c.DetachRolePolicy(ctx, roleName, policyName)
}

// convertPermissions 정책 문서를 GCP 권한 목록으로 변환
func convertPermissions(policy *csp.RolePolicy) []string {
	permissions := make(map[string]bool)
	for _, stmt := range policy.Statement {
		if stmt.Effect == "Allow" {
			for _, action := range stmt.Action {
				permissions[action] = true
			}
		}
	}

	result := make([]string, 0, len(permissions))
	for perm := range permissions {
		result = append(result, perm)
	}
	return result
}

// convertGCPTags GCP 역할 태그를 일반 태그로 변환
func convertGCPTags(role *adminpb.Role) map[string]string {
	tags := make(map[string]string)
	// GCP 역할에는 태그 기능이 없으므로 빈 맵 반환
	return tags
}
