package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCspRoleForMapping(t *testing.T) {
	cfgID := uint(1)
	okRole := &model.CspRole{ID: 1, Name: "mciam-admin", CspType: "aws", IdpIdentifier: testOidcProviderArn,
		IamIdentifier: "arn:aws:iam::111111111111:role/mciam-admin", CspIdpConfigID: &cfgID}
	oidcCfg := &model.CspIdpConfig{ID: cfgID, AuthMethod: model.AuthMethodOIDC}

	assert.NoError(t, validateCspRoleForMapping(okRole, oidcCfg, constants.AuthMethodOIDC, "aws"))
	assert.NoError(t, validateCspRoleForMapping(okRole, oidcCfg, "", ""), "authMethod 미지정은 OIDC, cspType 미지정은 검사하지 않음")

	assert.True(t, IsCspRoleRequestError(validateCspRoleForMapping(okRole, nil, constants.AuthMethodOIDC, "aws")), "소속 Config 없음")
	assert.True(t, IsCspRoleRequestError(validateCspRoleForMapping(okRole, oidcCfg, constants.AuthMethodSAML, "aws")), "인증 방식 불일치")
	assert.True(t, IsCspRoleRequestError(validateCspRoleForMapping(okRole, oidcCfg, constants.AuthMethodOIDC, "gcp")), "CSP 타입 불일치")

	noIdp := *okRole
	noIdp.IdpIdentifier = ""
	err := validateCspRoleForMapping(&noIdp, oidcCfg, constants.AuthMethodOIDC, "aws")
	require.True(t, IsCspRoleRequestError(err))
	assert.Contains(t, err.Error(), "idp_identifier is empty")

	noIam := *okRole
	noIam.IamIdentifier = ""
	err = validateCspRoleForMapping(&noIam, oidcCfg, constants.AuthMethodOIDC, "aws")
	require.True(t, IsCspRoleRequestError(err))
	assert.Contains(t, err.Error(), "iam_identifier is empty")
}

// TestValidateCspRoleForMapping_LoadsLinkedConfig: 서비스 메서드가 DB 의 소속 Config 를 읽어 검사한다.
func TestValidateCspRoleForMapping_LoadsLinkedConfig(t *testing.T) {
	svc := newTestCspRoleService(t)
	cfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, true, map[string]string{})
	linked := &model.CspRole{Name: "mciam-linked", CspType: "aws", IdpIdentifier: testOidcProviderArn, IamIdentifier: "arn:aws:iam::111111111111:role/mciam-linked"}
	applyCspIdpConfig(linked, cfg)
	require.NoError(t, svc.cspRoleRepo.CreateCspRoleRecord(linked))
	legacy := &model.CspRole{Name: "mciam-legacy", CspType: "aws", IdpIdentifier: testOidcProviderArn, IamIdentifier: "arn:aws:iam::111111111111:role/mciam-legacy"}
	require.NoError(t, svc.cspRoleRepo.CreateCspRoleRecord(legacy))

	assert.NoError(t, svc.ValidateCspRoleForMapping(linked.ID, constants.AuthMethodOIDC, "aws"))
	assert.True(t, IsCspRoleRequestError(svc.ValidateCspRoleForMapping(legacy.ID, constants.AuthMethodOIDC, "aws")))
}
