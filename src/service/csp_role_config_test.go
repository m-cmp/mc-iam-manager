package service

import (
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	mciamConfig "github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testOidcProviderArn = "arn:aws:iam::111111111111:oidc-provider/mcmp.example.com/auth/realms/mciam"

// seedIdpConfig CSP 계정과 그 계정의 IdP Config 를 만든다.
func seedIdpConfig(t *testing.T, db *gorm.DB, name, cspType string, authMethod model.AuthMethodType, active bool, cfg map[string]string) *model.CspIdpConfig {
	t.Helper()
	account := &model.CspAccount{Name: name + "-account", CspType: cspType, AccountInfo: map[string]string{"account_id": "111111111111"}}
	require.NoError(t, db.Create(account).Error)
	idpConfig := &model.CspIdpConfig{Name: name, CspAccountID: account.ID, AuthMethod: authMethod, Config: cfg, IsActive: true}
	require.NoError(t, db.Create(idpConfig).Error)
	if !active {
		// gorm 은 bool zero value 를 default:true 로 덮어쓰므로 생성 후 따로 끈다
		require.NoError(t, db.Model(idpConfig).Update("is_active", false).Error)
		idpConfig.IsActive = false
	}
	idpConfig.CspAccount = account
	return idpConfig
}

func uintPtr(v uint) *uint { return &v }

func TestResolveCspIdpConfig(t *testing.T) {
	oidcCfg := map[string]string{"role_arn": "arn:aws:iam::111111111111:role/mciam-platformadmin", "oidc_provider_arn": testOidcProviderArn}

	t.Run("지정한 Config 가 맞으면 그 Config", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		cfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, true, oidcCfg)
		got, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws", AuthMethod: constants.AuthMethodOIDC, CspIdpConfigID: uintPtr(cfg.ID)})
		require.NoError(t, err)
		assert.Equal(t, cfg.ID, got.ID)
	})

	t.Run("지정한 Config 가 없으면 요청 오류", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		_, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws", CspIdpConfigID: uintPtr(999)})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
	})

	t.Run("지정한 Config 가 비활성이면 요청 오류", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		cfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, false, oidcCfg)
		_, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws", CspIdpConfigID: uintPtr(cfg.ID)})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
	})

	t.Run("지정한 Config 의 CSP 타입·인증 방식이 다르면 요청 오류", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		gcpCfg := seedIdpConfig(t, svc.db, "gcp-oidc", "gcp", model.AuthMethodOIDC, true, map[string]string{})
		samlCfg := seedIdpConfig(t, svc.db, "aws-saml", "aws", model.AuthMethodSAML, true, map[string]string{})
		_, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws", CspIdpConfigID: uintPtr(gcpCfg.ID)})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
		// authMethod 미지정은 OIDC 로 본다
		_, err = svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws", CspIdpConfigID: uintPtr(samlCfg.ID)})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
	})

	t.Run("미지정이고 맞는 활성 Config 가 1개면 그 Config", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		cfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, true, oidcCfg)
		seedIdpConfig(t, svc.db, "aws-saml", "aws", model.AuthMethodSAML, true, map[string]string{})
		seedIdpConfig(t, svc.db, "aws-oidc-off", "aws", model.AuthMethodOIDC, false, oidcCfg)
		got, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspRoleName: "mciam-x", CspType: "aws"})
		require.NoError(t, err)
		assert.Equal(t, cfg.ID, got.ID)
	})

	t.Run("미지정이고 AWS 활성 Config 가 0개면 요청 오류", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		_, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws"})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
	})

	t.Run("미지정이고 AWS 활성 Config 가 2개 이상이면 요청 오류", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		seedIdpConfig(t, svc.db, "aws-oidc-a", "aws", model.AuthMethodOIDC, true, oidcCfg)
		seedIdpConfig(t, svc.db, "aws-oidc-b", "aws", model.AuthMethodOIDC, true, oidcCfg)
		_, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "aws"})
		assert.True(t, IsCspRoleRequestError(err), "%v", err)
	})

	t.Run("미지정이고 AWS 외 CSP 는 후보가 없으면 소속 없이 진행", func(t *testing.T) {
		svc := newTestCspRoleService(t)
		got, err := svc.resolveCspIdpConfig(&model.CreateCspRoleRequest{CspType: "gcp"})
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestCreateCspRole_GCP_SavesIdpConfig: AWS 외 CSP 도 소속 Config 와 그 계정을 저장한다.
func TestCreateCspRole_GCP_SavesIdpConfig(t *testing.T) {
	svc := newTestCspRoleService(t)
	cfg := seedIdpConfig(t, svc.db, "gcp-oidc", "gcp", model.AuthMethodOIDC, true, map[string]string{"audience": "//iam.googleapis.com/x"})

	role, err := svc.CreateCspRole(&model.CreateCspRoleRequest{
		CspRoleName:    "mcmp-admin",
		CspType:        "gcp",
		AuthMethod:     constants.AuthMethodOIDC,
		IdpIdentifier:  "//iam.googleapis.com/projects/1/locations/global/workloadIdentityPools/p/providers/mcmp",
		IamIdentifier:  "mcmp-admin@p.iam.gserviceaccount.com",
		CspIdpConfigID: uintPtr(cfg.ID),
	})
	require.NoError(t, err)

	saved, err := svc.cspRoleRepo.GetRoleByID(role.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.CspIdpConfigID)
	require.NotNil(t, saved.CspAccountID)
	assert.Equal(t, cfg.ID, *saved.CspIdpConfigID)
	assert.Equal(t, cfg.CspAccountID, *saved.CspAccountID)
}

// TestCreateCspRole_MismatchedIdpConfig_RequestError: 맞지 않는 Config 를 지정하면 레코드를 만들지 않고 요청 오류.
func TestCreateCspRole_MismatchedIdpConfig_RequestError(t *testing.T) {
	svc := newTestCspRoleService(t)
	awsCfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, true, map[string]string{})

	_, err := svc.CreateCspRole(&model.CreateCspRoleRequest{CspRoleName: "mcmp-admin", CspType: "gcp", CspIdpConfigID: uintPtr(awsCfg.ID)})
	assert.True(t, IsCspRoleRequestError(err), "%v", err)

	existing, err := svc.cspRoleRepo.GetCspRoleByName("mciam-mcmp-admin", "gcp")
	require.NoError(t, err)
	assert.Nil(t, existing)
}

func TestFederatedFromTrustPolicy(t *testing.T) {
	doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Federated":"` + testOidcProviderArn + `"},"Action":"sts:AssumeRoleWithWebIdentity"}]}`
	assert.Equal(t, []string{testOidcProviderArn}, federatedFromTrustPolicy(url.QueryEscape(doc)), "GetRole 은 URL 인코딩된 문서를 돌려준다")
	assert.Equal(t, []string{testOidcProviderArn}, federatedFromTrustPolicy(doc))

	other := "arn:aws:iam::111111111111:oidc-provider/other.example.com/auth/realms/mciam"
	multiDoc := `{"Statement":[{"Principal":{"Service":"ec2.amazonaws.com"}},{"Principal":{"Federated":"` + other + `"}},{"Principal":{"Federated":["` + testOidcProviderArn + `"]}}]}`
	assert.Equal(t, []string{other, testOidcProviderArn}, federatedFromTrustPolicy(multiDoc))

	assert.Empty(t, federatedFromTrustPolicy(""))
	assert.Empty(t, federatedFromTrustPolicy("not-json"))
	assert.Empty(t, federatedFromTrustPolicy(`{"Statement":[{"Principal":{"AWS":"arn:aws:iam::1:root"}}]}`))
}

func TestResolveIdpIdentifier(t *testing.T) {
	oidcCfg := &model.CspIdpConfig{AuthMethod: model.AuthMethodOIDC, Config: map[string]string{"oidc_provider_arn": testOidcProviderArn}}
	samlCfg := &model.CspIdpConfig{AuthMethod: model.AuthMethodSAML, Config: map[string]string{"saml_provider_arn": "arn:aws:iam::111111111111:saml-provider/keycloak"}}
	awsActual := []string{"arn:aws:iam::111111111111:oidc-provider/other.example.com/auth/realms/mciam", testOidcProviderArn}

	assert.Equal(t, "arn:req", resolveIdpIdentifier("r", "arn:req", oidcCfg, awsActual), "요청 값이 최우선 (신뢰 정책에 없으면 WARN)")
	assert.Equal(t, testOidcProviderArn, resolveIdpIdentifier("r", "", oidcCfg, awsActual), "다음은 소속 Config")
	assert.Equal(t, "arn:aws:iam::111111111111:saml-provider/keycloak", resolveIdpIdentifier("r", "", samlCfg, nil), "SAML 은 saml_provider_arn")
	assert.Equal(t, awsActual[0], resolveIdpIdentifier("r", "", nil, awsActual), "마지막은 AWS 실제 신뢰 정책의 첫 Federated")
	assert.Equal(t, awsActual[0], resolveIdpIdentifier("r", "", &model.CspIdpConfig{AuthMethod: model.AuthMethodOIDC}, awsActual), "Config 에 값이 없으면 AWS 값")
	assert.Equal(t, "", resolveIdpIdentifier("r", "", nil, nil))
}

// TestApplyAwsRoleToModel_PreservesStoredFields: polling·sync 로 AWS 값을 반영해도 소속·ExtendedConfig·idp_identifier 가 남는다.
func TestApplyAwsRoleToModel_PreservesStoredFields(t *testing.T) {
	cfgID, accountID := uint(7), uint(3)
	role := &model.CspRole{
		ID:             10,
		Name:           "mciam-admin",
		CspType:        "aws",
		IdpIdentifier:  testOidcProviderArn,
		CspIdpConfigID: &cfgID,
		CspAccountID:   &accountID,
		ExtendedConfig: map[string]interface{}{"saml_client_id": "urn:amazon:webservices"},
	}
	out := &iam.GetRoleOutput{Role: &iamtypes.Role{
		RoleName:   aws.String("mciam-admin"),
		Arn:        aws.String("arn:aws:iam::111111111111:role/mciam-admin"),
		RoleId:     aws.String("AROAEXAMPLE"),
		Path:       aws.String("/"),
		CreateDate: aws.Time(time.Now()),
	}}

	applyAwsRoleToModel(role, "", out)

	assert.Equal(t, uint(10), role.ID)
	assert.Equal(t, "created", role.Status)
	assert.Equal(t, "arn:aws:iam::111111111111:role/mciam-admin", role.IamIdentifier)
	assert.Equal(t, testOidcProviderArn, role.IdpIdentifier, "빈 idpIdentifier 로 기존 값을 지우지 않는다")
	assert.Equal(t, &cfgID, role.CspIdpConfigID)
	assert.Equal(t, &accountID, role.CspAccountID)
	assert.Equal(t, "urn:amazon:webservices", role.ExtendedConfig["saml_client_id"])
}

// TestUpdateCspRoleRecord_PreservesIdpConfigAcrossSave: 기존 레코드를 이어받아 Save 해도 소속이 DB 에 남는다(failed 재사용 경로).
func TestUpdateCspRoleRecord_PreservesIdpConfigAcrossSave(t *testing.T) {
	svc := newTestCspRoleService(t)
	cfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, true, map[string]string{"oidc_provider_arn": testOidcProviderArn})
	role := &model.CspRole{Name: "mciam-admin", CspType: "aws", Status: "failed"}
	applyCspIdpConfig(role, cfg)
	require.NoError(t, svc.cspRoleRepo.CreateCspRoleRecord(role))

	reloaded, err := svc.cspRoleRepo.GetCspRoleByName("mciam-admin", "aws")
	require.NoError(t, err)
	applyAwsRoleToModel(reloaded, testOidcProviderArn, &iam.GetRoleOutput{Role: &iamtypes.Role{
		RoleName: aws.String("mciam-admin"), Arn: aws.String("arn:aws:iam::111111111111:role/mciam-admin"),
	}})
	require.NoError(t, svc.cspRoleRepo.UpdateCspRoleRecord(reloaded))

	saved, err := svc.cspRoleRepo.GetRoleByID(role.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.CspIdpConfigID)
	assert.Equal(t, cfg.ID, *saved.CspIdpConfigID)
	assert.Equal(t, testOidcProviderArn, saved.IdpIdentifier)
	assert.Equal(t, "created", saved.Status)
}

func TestValidateAwsOidcTrustInputs(t *testing.T) {
	prevKC := mciamConfig.KC
	t.Cleanup(func() { mciamConfig.KC = prevKC })

	t.Setenv("MC_IAM_MANAGER_AWS_ACCOUNT_ID", "notyet")
	t.Setenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID", "mciam-oidc-Client")
	mciamConfig.KC = &mciamConfig.KeycloakConfig{ExternalURL: "https://mcmp.example.com/auth"}
	assert.True(t, IsCspRoleRequestError(validateAwsOidcTrustInputs()), "샘플 계정 ID")

	t.Setenv("MC_IAM_MANAGER_AWS_ACCOUNT_ID", "111111111111")
	t.Setenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID", "notyet")
	assert.True(t, IsCspRoleRequestError(validateAwsOidcTrustInputs()), "샘플 audience")

	t.Setenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID", "mciam-oidc-Client")
	for _, host := range []string{"", "http://mc-iam-manager-kc:8080/auth", "http://localhost:8080/auth", "http://127.0.0.1/auth"} {
		mciamConfig.KC = &mciamConfig.KeycloakConfig{ExternalURL: host}
		assert.True(t, IsCspRoleRequestError(validateAwsOidcTrustInputs()), "내부 호스트 %q", host)
	}

	mciamConfig.KC = &mciamConfig.KeycloakConfig{ExternalURL: "https://mcmp.example.com/auth"}
	assert.NoError(t, validateAwsOidcTrustInputs())
}
