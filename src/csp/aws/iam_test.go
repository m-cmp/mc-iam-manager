package aws

import (
	"context"
	"testing"
	"time"

	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// WorkspaceRole 테스트용 워크스페이스 역할 모델
type WorkspaceRole struct {
	ID          uint   `gorm:"primaryKey"`
	WorkspaceID string `gorm:"column:workspace_id"`
}

// WorkspaceRoleCSPRoleMapping 테스트용 워크스페이스 역할-CSP 역할 매핑 모델
type WorkspaceRoleCSPRoleMapping struct {
	ID              uint   `gorm:"primaryKey"`
	WorkspaceRoleID uint   `gorm:"column:workspace_role_id"`
	CSPType         string `gorm:"column:csp_type"`
	RoleARN         string `gorm:"column:role_arn"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 테스트용 테이블 생성
	err = db.AutoMigrate(&WorkspaceRole{}, &WorkspaceRoleCSPRoleMapping{})
	require.NoError(t, err)

	// 테스트 데이터 삽입
	workspaceRole := &WorkspaceRole{
		ID:          1,
		WorkspaceID: "e2464ed3-8d96-4c5e-84ad-22c22de38576", // JWT의 sub 값과 일치
	}
	err = db.Create(workspaceRole).Error
	require.NoError(t, err)

	cspRoleMapping := &WorkspaceRoleCSPRoleMapping{
		ID:              1,
		WorkspaceRoleID: 1,
		CSPType:         "AWS",
		RoleARN:         "arn:aws:iam::050864702683:role/mciam_viewer",
	}
	err = db.Create(cspRoleMapping).Error
	require.NoError(t, err)

	return db
}

func TestAWSIAMClient(t *testing.T) {
	// 테스트 DB 설정
	db := setupTestDB(t)

	// 테스트 설정
	config := &csp.IAMClientConfig{
		Region:           "ap-northeast-2",
		WebIdentityToken: "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJmRmtqcWltT1FhSEs0ZlQ2YWVlV3VZUi1KOFk3TjdEWmV2ZFJQZjRNNmNJIn0.eyJleHAiOjE3NDczOTU4OTAsImlhdCI6MTc0NzM1OTg5NSwianRpIjoiNjI0OGY0NzYtZDU4MC00MGFjLWEyZDgtMTE5MzNjZGNiZmMwIiwiaXNzIjoiaHR0cHM6Ly9tY2lhbS5vbmVjbG91ZGNvbi5jb20vcmVhbG1zL21jaWFtLWRlbXAzIiwiYXVkIjoib2lkYy1zZXJ2aWNlQWNjb3VudENsaWVudCIsInN1YiI6ImEzNDQ1Y2VjLTBjZjQtNGFkNi1hNjk0LTUyMDdjYmMxMWIzOSIsInR5cCI6IkJlYXJlciIsImF6cCI6Im1jaWFtQ2xpZW50Iiwic2Vzc2lvbl9zdGF0ZSI6ImQzNDE1Y2U3LTA0MjItNGQ1ZS1iNTlhLWVhMDIzNmI5Njk4ZCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiLyoiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIm9mZmxpbmVfYWNjZXNzIiwidW1hX2F1dGhvcml6YXRpb24iLCJkZWZhdWx0LXJvbGVzLW1jaWFtLWRlbXAzIiwicGxhdGZvcm1BZG1pbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1jaWFtLW9pZGMtY2xpZW50LWRlbXAzIjp7InJvbGVzIjpbIm9pZGMtYXNzdW1lLXJvbGUiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sImF1dGhvcml6YXRpb24iOnsicGVybWlzc2lvbnMiOlt7InJzaWQiOiIwMTVmZTBhNC0xZjA0LTQzY2UtYWE4Yy00N2NiMjg5ZWVhNjMiLCJyc25hbWUiOiJEZWZhdWx0IFJlc291cmNlIn1dfSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDM0MTVjZTctMDQyMi00ZDVlLWI1OWEtZWEwMjM2Yjk2OThkIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsIm5hbWUiOiJsZWUgbWFuIiwicHJlZmVycmVkX3VzZXJuYW1lIjoibGVlbWFuIiwiZ2l2ZW5fbmFtZSI6ImxlZSIsImZhbWlseV9uYW1lIjoibWFuIiwiZW1haWwiOiJsZWVtYW5AdGVzdC5jb20ifQ.WZPH_0VvUhJNm7S5E21BFnK0kfH30Wevheoy9IvZtmUR-vWA4-8Jv7Ilu_o9BINJm47GNodK71R-PxC3MfTwjFJ7apT-m5r2AKtrzvIUE7HFuzaykcrqQ2kmBnUHqguFcSoo0GlZ7vfqO9fYMv5Gyn62bKk25ITleItahTFaY198M0p0je_xwLbm0wf7Wj7WfvqHVIyNgIZetksoe00ikNXCHeLjU1lLTPO-zT-QvT46CV0ETnPO9tIRiPSe2utP96GQ1Tav2YIW9zXE-hGb2pGWDCA5AK-ohABoOMedwm9ajReDDclGKE0SUAePs2GP1eAcke8IJ3NcGHTPyhClww",
		WorkspaceTicket:  "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJmRmtqcWltT1FhSEs0ZlQ2YWVlV3VZUi1KOFk3TjdEWmV2ZFJQZjRNNmNJIn0.eyJleHAiOjE3NDczOTU4OTAsImlhdCI6MTc0NzM1OTg5NSwianRpIjoiNjI0OGY0NzYtZDU4MC00MGFjLWEyZDgtMTE5MzNjZGNiZmMwIiwiaXNzIjoiaHR0cHM6Ly9tY2lhbS5vbmVjbG91ZGNvbi5jb20vcmVhbG1zL21jaWFtLWRlbXAzIiwiYXVkIjoib2lkYy1zZXJ2aWNlQWNjb3VudENsaWVudCIsInN1YiI6ImEzNDQ1Y2VjLTBjZjQtNGFkNi1hNjk0LTUyMDdjYmMxMWIzOSIsInR5cCI6IkJlYXJlciIsImF6cCI6Im1jaWFtQ2xpZW50Iiwic2Vzc2lvbl9zdGF0ZSI6ImQzNDE1Y2U3LTA0MjItNGQ1ZS1iNTlhLWVhMDIzNmI5Njk4ZCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiLyoiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIm9mZmxpbmVfYWNjZXNzIiwidW1hX2F1dGhvcml6YXRpb24iLCJkZWZhdWx0LXJvbGVzLW1jaWFtLWRlbXAzIiwicGxhdGZvcm1BZG1pbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1jaWFtLW9pZGMtY2xpZW50LWRlbXAzIjp7InJvbGVzIjpbIm9pZGMtYXNzdW1lLXJvbGUiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sImF1dGhvcml6YXRpb24iOnsicGVybWlzc2lvbnMiOlt7InJzaWQiOiIwMTVmZTBhNC0xZjA0LTQzY2UtYWE4Yy00N2NiMjg5ZWVhNjMiLCJyc25hbWUiOiJEZWZhdWx0IFJlc291cmNlIn1dfSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDM0MTVjZTctMDQyMi00ZDVlLWI1OWEtZWEwMjM2Yjk2OThkIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsIm5hbWUiOiJsZWUgbWFuIiwicHJlZmVycmVkX3VzZXJuYW1lIjoibGVlbWFuIiwiZ2l2ZW5fbmFtZSI6ImxlZSIsImZhbWlseV9uYW1lIjoibWFuIiwiZW1haWwiOiJsZWVtYW5AdGVzdC5jb20ifQ.WZPH_0VvUhJNm7S5E21BFnK0kfH30Wevheoy9IvZtmUR-vWA4-8Jv7Ilu_o9BINJm47GNodK71R-PxC3MfTwjFJ7apT-m5r2AKtrzvIUE7HFuzaykcrqQ2kmBnUHqguFcSoo0GlZ7vfqO9fYMv5Gyn62bKk25ITleItahTFaY198M0p0je_xwLbm0wf7Wj7WfvqHVIyNgIZetksoe00ikNXCHeLjU1lLTPO-zT-QvT46CV0ETnPO9tIRiPSe2utP96GQ1Tav2YIW9zXE-hGb2pGWDCA5AK-ohABoOMedwm9ajReDDclGKE0SUAePs2GP1eAcke8IJ3NcGHTPyhClww",
		Timeout:          300 * time.Second,
	}

	// 테스트 데이터 수정
	// cspRoleMapping := &WorkspaceRoleCSPRoleMapping{
	// 	ID:              1,
	// 	WorkspaceRoleID: 1,
	// 	CSPType:         "AWS",
	// 	RoleARN:         "arn:aws:iam::050864702683:role/mciam_viewer",
	// }
	//err := db.Model(&WorkspaceRoleCSPRoleMapping{}).Where("id = ?", 1).Updates(cspRoleMapping).Error
	//require.NoError(t, err)

	// 클라이언트 생성
	client, err := NewAWSIAMClient(config, db)
	require.NoError(t, err)
	require.NotNil(t, client)

	// 테스트용 역할 정보
	// testRole := &csp.Role{
	// 	Name:        "mciam_viewer_test",
	// 	Description: "Test role for IAM manager",
	// 	Policy: &csp.RolePolicy{
	// 		Version: "2012-10-17",
	// 		Statement: []csp.RolePolicyStatement{
	// 			{
	// 				Effect: "Allow",
	// 				Action: []string{
	// 					"ec2:Describe*",
	// 					"ec2:Get*",
	// 					"ec2:List*",
	// 				},
	// 				Resource: []string{"*"},
	// 				Principal: &csp.Principal{
	// 					Service: []string{"ec2.amazonaws.com"},
	// 				},
	// 			},
	// 		},
	// 	},
	// 	Tags: map[string]string{
	// 		"Environment": "test",
	// 		"Project":     "iam-manager",
	// 	},
	// }

	ctx := context.Background()

	// t.Run("CreateRole", func(t *testing.T) {
	// 	err := client.CreateRole(ctx, testRole)
	// 	assert.NoError(t, err)
	// })

	t.Run("GetRole", func(t *testing.T) {
		//role, err := client.GetRole(ctx, testRole.Name)
		//role, err := client.GetRole(ctx, "mciam_viewer")
		role, err := client.GetRole(ctx, "mciam-csp-role-manager")
		assert.NoError(t, err)
		assert.NotNil(t, role)
		// assert.Equal(t, testRole.Name, role.Name)
		// assert.Equal(t, testRole.Description, role.Description)
		// assert.Equal(t, testRole.Tags, role.Tags)
	})

	// t.Run("UpdateRole", func(t *testing.T) {
	// 	// 역할 정보 수정
	// 	testRole.Description = "Updated test role description"
	// 	testRole.Policy.Statement[0].Action = append(testRole.Policy.Statement[0].Action, "s3:PutObject")

	// 	err := client.UpdateRole(ctx, testRole)
	// 	assert.NoError(t, err)

	// 	// 수정된 역할 정보 확인
	// 	role, err := client.GetRole(ctx, testRole.Name)
	// 	assert.NoError(t, err)
	// 	assert.NotNil(t, role)
	// 	assert.Equal(t, testRole.Description, role.Description)
	// 	assert.Contains(t, role.Policy.Statement[0].Action, "s3:PutObject")
	// })

	// t.Run("AttachRolePolicy", func(t *testing.T) {
	// 	policyArn := "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"
	// 	err := client.AttachRolePolicy(ctx, testRole.Name, policyArn)
	// 	assert.NoError(t, err)

	// 	// 연결된 정책 목록 확인
	// 	policies, err := client.ListRolePolicies(ctx, testRole.Name)
	// 	assert.NoError(t, err)
	// 	assert.Contains(t, policies, "AmazonS3ReadOnlyAccess")
	// })

	// t.Run("DetachRolePolicy", func(t *testing.T) {
	// 	policyArn := "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"
	// 	err := client.DetachRolePolicy(ctx, testRole.Name, policyArn)
	// 	assert.NoError(t, err)

	// 	// 정책이 제거되었는지 확인
	// 	policies, err := client.ListRolePolicies(ctx, testRole.Name)
	// 	assert.NoError(t, err)
	// 	assert.NotContains(t, policies, "AmazonS3ReadOnlyAccess")
	// })

	// t.Run("DeleteRole", func(t *testing.T) {
	// 	err := client.DeleteRole(ctx, testRole.Name)
	// 	assert.NoError(t, err)

	// 	// 역할이 삭제되었는지 확인
	// 	_, err = client.GetRole(ctx, testRole.Name)
	// 	assert.Error(t, err)
	// })
}
