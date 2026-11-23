package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IAM-BUG-051: POST /roles/csp 는 AWS 레코드가 DB 에 있어도 GetRole 로 다시 확인한다
func TestCreateCspRole_ExistingAwsRecordRechecked(t *testing.T) {
	svc, api, cfg := newTargetTestService(t, nil)
	stale := &model.CspRole{Name: "mciam-t045-stale", CspType: "aws", Status: "created", IamRoleId: "AROA-OLD", CspIdpConfigID: &cfg.ID}
	require.NoError(t, svc.db.Create(stale).Error)

	t.Run("AWS 에 있으면 값을 다시 동기화", func(t *testing.T) {
		api.roles["mciam-t045-stale"] = fakeAwsRole("mciam-t045-stale", `{"Statement":[{"Principal":{"Federated":"`+testOidcProviderArn+`"}}]}`)
		got, err := svc.CreateCspRole(&model.CreateCspRoleRequest{CspRoleName: "mciam-t045-stale", CspType: "aws"})
		require.NoError(t, err)
		assert.Equal(t, stale.ID, got.ID, "같은 레코드")
		assert.Equal(t, "AROAmciam-t045-stale", got.IamRoleId, "AWS 값으로 갱신")
		assert.Empty(t, api.createCalls)
	})
	t.Run("AWS 에서 지워졌으면 소속 Config 기준으로 다시 만든다", func(t *testing.T) {
		delete(api.roles, "mciam-t045-stale")
		got, err := svc.CreateCspRole(&model.CreateCspRoleRequest{CspRoleName: "mciam-t045-stale", CspType: "aws"})
		require.NoError(t, err)
		assert.Equal(t, stale.ID, got.ID)
		require.Len(t, api.createCalls, 1, "stale 레코드를 그대로 돌려주지 않는다")
		assert.Equal(t, "created", got.Status)
	})
}

// IAM-BUG-052: AWS 에 이미 없는 Role 이면 DB 레코드만 지운다
func TestDeleteCspRole_AwsNoSuchEntityRemovesRecord(t *testing.T) {
	svc, api, cfg := newTargetTestService(t, nil)
	orphan := &model.CspRole{Name: "mciam-t045-orphan", CspType: "aws", Status: "created", CspIdpConfigID: &cfg.ID}
	require.NoError(t, svc.db.Create(orphan).Error)

	require.NoError(t, svc.DeleteCspRole(orphan.ID))
	assert.Equal(t, []string{"mciam-t045-orphan"}, api.deleteCalls)
	_, err := svc.cspRoleRepo.GetRoleByID(orphan.ID)
	assert.Error(t, err, "레코드가 지워져야 한다")

	// AWS 에 있으면 AWS 와 DB 모두 지운다
	live := &model.CspRole{Name: "mciam-t045-live", CspType: "aws", Status: "created", CspIdpConfigID: &cfg.ID}
	require.NoError(t, svc.db.Create(live).Error)
	api.roles["mciam-t045-live"] = fakeAwsRole("mciam-t045-live", "{}")
	require.NoError(t, svc.DeleteCspRole(live.ID))
	_, ok := api.roles["mciam-t045-live"]
	assert.False(t, ok)
}
