package service

import (
	"testing"

	"github.com/aws/smithy-go"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 관리용 Role 보호: 이름·ARN 판정, CspRole 등록·수정·삭제 409
func TestGuardNotManagementRole(t *testing.T) {
	svc, _, _ := newTargetTestService(t, nil)
	assert.True(t, IsCspRoleProtectedError(svc.GuardNotManagementRole("mciam-platformadmin", "", "mapped")), "이름으로 판정")
	assert.True(t, IsCspRoleProtectedError(svc.GuardNotManagementRole("", testManagementRoleArn, "mapped")), "ARN 으로 판정")
	assert.NoError(t, svc.GuardNotManagementRole("mciam-t045-x", "arn:aws:iam::111111111111:role/mciam-t045-x", "mapped"))

	// 비활성 Config 의 관리용 Role 도 보호
	seedIdpConfig(t, svc.db, "aws-old", "aws", model.AuthMethodOIDC, false, map[string]string{"role_arn": "arn:aws:iam::111111111111:role/mciam-admin"})
	assert.True(t, IsCspRoleProtectedError(svc.GuardNotManagementRole("mciam-admin", "", "mapped")))
}

func TestCspRoleCRUD_ManagementRoleProtected(t *testing.T) {
	svc, api, _ := newTargetTestService(t, nil)

	// 등록: 관리용 Role 이름·ARN 으로 POST /roles/csp → 409 성격, AWS 호출 없음
	_, err := svc.CreateCspRole(&model.CreateCspRoleRequest{CspRoleName: "mciam-platformadmin", CspType: "aws"})
	assert.True(t, IsCspRoleProtectedError(err), "%v", err)
	_, err = svc.CreateCspRole(&model.CreateCspRoleRequest{CspRoleName: "platformadmin", CspType: "aws"})
	assert.True(t, IsCspRoleProtectedError(err), "접두사 정규화 후 판정: %v", err)
	assert.Zero(t, api.getCalls)
	assert.Zero(t, countCspRoles(t, svc))

	// 기존 설치에 남은 관리용 Role 레코드(부트스트랩 9-3)는 수정·삭제 409
	legacy := &model.CspRole{Name: "mciam-platformadmin", CspType: "aws", Status: "created", IamIdentifier: testManagementRoleArn}
	require.NoError(t, svc.db.Create(legacy).Error)
	assert.True(t, IsCspRoleProtectedError(svc.UpdateCspRole(legacy.ID, &model.CreateCspRoleRequest{Description: "x"})))
	assert.True(t, IsCspRoleProtectedError(svc.DeleteCspRole(legacy.ID)), "AWS DeleteRole 전에 막는다")
	var still model.CspRole
	require.NoError(t, svc.db.First(&still, legacy.ID).Error)

	// 다른 레코드를 관리용 Role 로 바꾸는 수정도 409
	other := &model.CspRole{Name: "mciam-t045-other", CspType: "aws", Status: "created"}
	require.NoError(t, svc.db.Create(other).Error)
	assert.True(t, IsCspRoleProtectedError(svc.UpdateCspRole(other.ID, &model.CreateCspRoleRequest{IamIdentifier: testManagementRoleArn})))
	assert.NoError(t, svc.UpdateCspRole(other.ID, &model.CreateCspRoleRequest{Description: "ok"}))
}

// IAM-TECH-050: CreateRole 전에 OIDC provider 존재·audience 확인
func TestEnsureAwsRole_VerifiesOidcProvider(t *testing.T) {
	t.Run("provider 없음 → 요청 오류, 레코드·CreateRole 없음", func(t *testing.T) {
		svc, api, _ := newTargetTestService(t, nil)
		api.providers = map[string][]string{}
		_, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-noprov", CspType: "aws"})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
		assert.Contains(t, err.Error(), "does not exist")
		assert.Empty(t, api.createCalls)
		assert.Zero(t, countCspRoles(t, svc))
	})
	t.Run("audience 가 ClientIDList 에 없음 → 요청 오류", func(t *testing.T) {
		svc, api, _ := newTargetTestService(t, nil)
		api.providers[testOidcProviderArn] = []string{"other-client"}
		_, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-noaud", CspType: "aws"})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
		assert.Contains(t, err.Error(), "client ID list")
		assert.Empty(t, api.createCalls)
	})
	t.Run("권한 없음(AccessDenied) → 건너뛰고 생성", func(t *testing.T) {
		svc, api, _ := newTargetTestService(t, nil)
		api.providerErr = &smithy.GenericAPIError{Code: "AccessDenied", Message: "not authorized"}
		_, result, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-denied", CspType: "aws"})
		require.NoError(t, err)
		assert.Equal(t, TargetResultCreatedAndMapped, result)
		assert.Len(t, api.createCalls, 1)
	})
	t.Run("AWS 에 이미 있는 Role 등록은 provider 를 확인하지 않는다", func(t *testing.T) {
		svc, api, _ := newTargetTestService(t, nil)
		api.providers = map[string][]string{}
		api.roles["mciam-t045-pre"] = fakeAwsRole("mciam-t045-pre", `{"Statement":[{"Principal":{"Federated":"`+testOidcProviderArn+`"}}]}`)
		_, result, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-pre", CspType: "aws"})
		require.NoError(t, err)
		assert.Equal(t, TargetResultRegisteredAndMapped, result)
	})
	t.Run("check 는 생성 불가 사유를 message 로 알린다", func(t *testing.T) {
		svc, api, _ := newTargetTestService(t, nil)
		api.providers = map[string][]string{}
		got := svc.CheckTargetCspRole(model.CreateCspRoleRequest{CspRoleName: "t045-x", CspType: "aws"})
		assert.Equal(t, TargetCheckNotFound, got.Status)
		assert.Contains(t, got.Message, "does not exist")
	})
}
