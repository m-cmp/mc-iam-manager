package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCspRoleService(t *testing.T) *CspRoleService {
	db := setupTestDB(t)
	return NewCspRoleService(db, &mockKeycloakService{})
}

// TestCreateCspRole_GCP_PersistsIdentifiers: 비-AWS CSP(default 분기)도 요청의
// idpIdentifier/iamIdentifier를 그대로 저장해야 한다 (OI-24 회귀 방지).
func TestCreateCspRole_GCP_PersistsIdentifiers(t *testing.T) {
	svc := newTestCspRoleService(t)

	req := &model.CreateCspRoleRequest{
		CspRoleName:   "mcmp-admin",
		CspType:       "gcp",
		AuthMethod:    constants.AuthMethodOIDC,
		IdpIdentifier: "//iam.googleapis.com/projects/295058475885/locations/global/workloadIdentityPools/mcmp-oidc/providers/mcmp",
		IamIdentifier: "mcmp-admin@csta-349809.iam.gserviceaccount.com",
	}

	role, err := svc.CreateCspRole(req)
	require.NoError(t, err)
	require.NotNil(t, role)

	assert.Equal(t, req.IdpIdentifier, role.IdpIdentifier)
	assert.Equal(t, req.IamIdentifier, role.IamIdentifier)
	assert.Equal(t, "created", role.Status)
}

// TestCreateCspRole_Alibaba_PersistsIdentifiers: 동일한 default 분기를 타는
// 다른 CSP(alibaba)에서도 identifier가 저장되는지 확인.
func TestCreateCspRole_Alibaba_PersistsIdentifiers(t *testing.T) {
	svc := newTestCspRoleService(t)

	req := &model.CreateCspRoleRequest{
		CspRoleName:   "mciam-oidc-role",
		CspType:       "alibaba",
		AuthMethod:    constants.AuthMethodOIDC,
		IdpIdentifier: "acs:ram::5513479151634744:oidc-provider/mciam-keycloak",
		IamIdentifier: "acs:ram::5513479151634744:role/mciam-oidc-role",
	}

	role, err := svc.CreateCspRole(req)
	require.NoError(t, err)
	require.NotNil(t, role)

	assert.Equal(t, req.IdpIdentifier, role.IdpIdentifier)
	assert.Equal(t, req.IamIdentifier, role.IamIdentifier)
}
