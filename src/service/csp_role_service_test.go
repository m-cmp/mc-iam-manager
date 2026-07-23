package service

import (
	"encoding/json"
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

// TestGetAwsSamlAssumeRolePolicyDocument_Success: AWS SAML CspRole 생성 시
// trust policy가 SAML Provider ARN을 Federated principal로, sts:AssumeRoleWithSAML을
// Action으로 갖는지 확인한다 (기존 OIDC 전용 getAwsAssumeRolePolicyDocument와 별도 경로).
func TestGetAwsSamlAssumeRolePolicyDocument_Success(t *testing.T) {
	req := &model.CreateCspRoleRequest{
		CspRoleName:   "mciam-admin-saml",
		CspType:       "aws",
		AuthMethod:    constants.AuthMethodSAML,
		IdpIdentifier: "arn:aws:iam::050864702683:saml-provider/mcmp-dev-cscmzc-com-saml",
	}

	doc, err := getAwsSamlAssumeRolePolicyDocument(req)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(doc), &parsed))

	statements := parsed["Statement"].([]interface{})
	require.Len(t, statements, 1)
	statement := statements[0].(map[string]interface{})

	assert.Equal(t, "sts:AssumeRoleWithSAML", statement["Action"])
	principal := statement["Principal"].(map[string]interface{})
	assert.Equal(t, req.IdpIdentifier, principal["Federated"])
	condition := statement["Condition"].(map[string]interface{})
	stringEquals := condition["StringEquals"].(map[string]interface{})
	assert.Equal(t, "https://signin.aws.amazon.com/saml", stringEquals["SAML:aud"])
}

// TestGetAwsSamlAssumeRolePolicyDocument_MissingIdpIdentifier: SAML Provider ARN
// 없이는 trust policy를 만들 수 없으므로 명시적으로 에러를 반환해야 한다.
func TestGetAwsSamlAssumeRolePolicyDocument_MissingIdpIdentifier(t *testing.T) {
	req := &model.CreateCspRoleRequest{
		CspRoleName: "mciam-admin-saml",
		CspType:     "aws",
		AuthMethod:  constants.AuthMethodSAML,
	}

	_, err := getAwsSamlAssumeRolePolicyDocument(req)
	assert.Error(t, err)
}

// TestUpdateCspRole_PersistsExtendedConfig: SAML 발급에 필수인
// ExtendedConfig["saml_client_id"]를 raw SQL이 아니라 UpdateCspRole API로
// 설정할 수 있어야 한다.
func TestUpdateCspRole_PersistsExtendedConfig(t *testing.T) {
	svc := newTestCspRoleService(t)

	created, err := svc.CreateCspRole(&model.CreateCspRoleRequest{
		CspRoleName:   "mcmp-admin-extcfg",
		CspType:       "gcp",
		AuthMethod:    constants.AuthMethodOIDC,
		IdpIdentifier: "//iam.googleapis.com/projects/x/locations/global/workloadIdentityPools/p/providers/pr",
		IamIdentifier: "sa@project.iam.gserviceaccount.com",
	})
	require.NoError(t, err)

	err = svc.UpdateCspRole(created.ID, &model.CreateCspRoleRequest{
		CspRoleName: created.Name,
		ExtendedConfig: map[string]interface{}{
			"saml_client_id": "urn:amazon:webservices",
		},
	})
	require.NoError(t, err)

	updated, err := svc.GetCspRoleByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "urn:amazon:webservices", updated.ExtendedConfig["saml_client_id"])
}
