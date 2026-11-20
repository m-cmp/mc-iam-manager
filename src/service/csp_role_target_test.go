package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAwsRoleAPI 대상 CSP Role 확정 테스트용 AWS IAM
type fakeAwsRoleAPI struct {
	roles       map[string]*iamtypes.Role
	getErr      error // NoSuchEntity 외의 GetRole 오류
	createErr   error
	getCalls    int
	createCalls []*iam.CreateRoleInput
}

func newFakeAwsRoleAPI() *fakeAwsRoleAPI {
	return &fakeAwsRoleAPI{roles: map[string]*iamtypes.Role{}}
}

func fakeAwsRole(name, trustDoc string) *iamtypes.Role {
	now := time.Now()
	return &iamtypes.Role{
		RoleName:                 aws.String(name),
		Arn:                      aws.String("arn:aws:iam::111111111111:role/" + name),
		RoleId:                   aws.String("AROA" + name),
		Path:                     aws.String("/"),
		CreateDate:               &now,
		AssumeRolePolicyDocument: aws.String(url.QueryEscape(trustDoc)),
	}
}

func (f *fakeAwsRoleAPI) GetRole(_ context.Context, in *iam.GetRoleInput, _ ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if r, ok := f.roles[aws.ToString(in.RoleName)]; ok {
		return &iam.GetRoleOutput{Role: r}, nil
	}
	return nil, &iamtypes.NoSuchEntityException{Message: aws.String("not found")}
}

func (f *fakeAwsRoleAPI) CreateRole(_ context.Context, in *iam.CreateRoleInput, _ ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
	f.createCalls = append(f.createCalls, in)
	if f.createErr != nil {
		return nil, f.createErr
	}
	name := aws.ToString(in.RoleName)
	f.roles[name] = fakeAwsRole(name, aws.ToString(in.AssumeRolePolicyDocument))
	return &iam.CreateRoleOutput{Role: f.roles[name]}, nil
}

const testManagementRoleArn = "arn:aws:iam::111111111111:role/mciam-platformadmin"

func newTargetTestService(t *testing.T, cfgValues map[string]string) (*CspRoleService, *fakeAwsRoleAPI, *model.CspIdpConfig) {
	t.Helper()
	prev := awsRolePollInterval
	awsRolePollInterval = time.Millisecond
	t.Cleanup(func() { awsRolePollInterval = prev })

	svc := newTestCspRoleService(t)
	if cfgValues == nil {
		cfgValues = map[string]string{"role_arn": testManagementRoleArn, "oidc_provider_arn": testOidcProviderArn, "audience": "mciam-oidc-Client"}
	}
	cfg := seedIdpConfig(t, svc.db, "aws-oidc", "aws", model.AuthMethodOIDC, true, cfgValues)
	api := newFakeAwsRoleAPI()
	svc.awsRoleClient = func(*model.CspIdpConfig) (awsRoleAPI, error) { return api, nil }
	return svc, api, cfg
}

func countCspRoles(t *testing.T, svc *CspRoleService) int64 {
	t.Helper()
	var n int64
	require.NoError(t, svc.db.Model(&model.CspRole{}).Count(&n).Error)
	return n
}

func TestNormalizeTargetCspRoleName(t *testing.T) {
	_, err := NormalizeTargetCspRoleName("  ")
	assert.True(t, IsCspRoleRequestError(err), "빈 이름은 요청 오류 (자동 명명 없음)")

	got, err := NormalizeTargetCspRoleName(" t045-new ")
	require.NoError(t, err)
	assert.Equal(t, "mciam-t045-new", got)

	got, err = NormalizeTargetCspRoleName("mciam-t045-pre")
	require.NoError(t, err)
	assert.Equal(t, "mciam-t045-pre", got, "이미 접두사가 있으면 그대로")
}

func TestBuildAwsOidcTrustPolicy(t *testing.T) {
	cfg := &model.CspIdpConfig{ID: 1, Name: "c", Config: map[string]string{"oidc_provider_arn": testOidcProviderArn, "audience": "mciam-oidc-Client"}}
	doc, err := buildAwsOidcTrustPolicy(cfg)
	require.NoError(t, err)
	var policy struct {
		Statement []struct {
			Principal map[string]string
			Action    string
			Condition map[string]map[string]string
		}
	}
	require.NoError(t, json.Unmarshal([]byte(doc), &policy))
	require.Len(t, policy.Statement, 1)
	st := policy.Statement[0]
	assert.Equal(t, testOidcProviderArn, st.Principal["Federated"])
	assert.Equal(t, "sts:AssumeRoleWithWebIdentity", st.Action)
	assert.Equal(t, "mciam-oidc-Client", st.Condition["StringEquals"]["mcmp.example.com/auth/realms/mciam:aud"])

	for name, values := range map[string]map[string]string{
		"provider 없음":    {"audience": "a"},
		"provider 형식 오류": {"oidc_provider_arn": "mcmp.example.com", "audience": "a"},
		"audience 없음":    {"oidc_provider_arn": testOidcProviderArn},
		"audience 샘플값":   {"oidc_provider_arn": testOidcProviderArn, "audience": "notyet"},
	} {
		_, err := buildAwsOidcTrustPolicy(&model.CspIdpConfig{Config: values})
		assert.True(t, IsCspRoleRequestError(err), name)
	}
}

func TestResolveTargetCspRole_CreatesWithConfigTrustPolicy(t *testing.T) {
	svc, api, cfg := newTargetTestService(t, nil)
	t.Setenv("MC_IAM_MANAGER_AWS_ACCOUNT_ID", "notyet") // env 는 더 이상 쓰지 않는다
	t.Setenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID", "notyet")

	role, result, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-new", CspType: "aws"})
	require.NoError(t, err)
	assert.Equal(t, TargetResultCreatedAndMapped, result)
	require.Len(t, api.createCalls, 1)
	assert.Equal(t, "mciam-t045-new", aws.ToString(api.createCalls[0].RoleName))
	assert.Contains(t, aws.ToString(api.createCalls[0].AssumeRolePolicyDocument), testOidcProviderArn)
	assert.Contains(t, aws.ToString(api.createCalls[0].AssumeRolePolicyDocument), "mciam-oidc-Client")

	saved, err := svc.cspRoleRepo.GetRoleByID(role.ID)
	require.NoError(t, err)
	assert.Equal(t, "created", saved.Status)
	assert.Equal(t, "arn:aws:iam::111111111111:role/mciam-t045-new", saved.IamIdentifier)
	assert.Equal(t, testOidcProviderArn, saved.IdpIdentifier)
	require.NotNil(t, saved.CspIdpConfigID)
	assert.Equal(t, cfg.ID, *saved.CspIdpConfigID)
}

func TestResolveTargetCspRole_RegistersExistingAwsRole(t *testing.T) {
	svc, api, _ := newTargetTestService(t, nil)
	api.roles["mciam-t045-pre"] = fakeAwsRole("mciam-t045-pre", `{"Statement":[{"Principal":{"Federated":"`+testOidcProviderArn+`"}}]}`)

	role, result, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "mciam-t045-pre", CspType: "aws"})
	require.NoError(t, err)
	assert.Equal(t, TargetResultRegisteredAndMapped, result)
	assert.Empty(t, api.createCalls, "AWS 에 있으면 CreateRole 을 부르지 않는다")
	assert.Equal(t, "arn:aws:iam::111111111111:role/mciam-t045-pre", role.IamIdentifier)

	// 두 번째 저장은 DB 레코드를 그대로 쓴다 (AWS 조회 없음)
	calls := api.getCalls
	again, result, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-pre", CspType: "aws"})
	require.NoError(t, err)
	assert.Equal(t, TargetResultMapped, result)
	assert.Equal(t, role.ID, again.ID)
	assert.Equal(t, calls, api.getCalls)
}

func TestResolveTargetCspRole_ManagementRoleProtected(t *testing.T) {
	svc, api, _ := newTargetTestService(t, nil)
	protectedRole, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "mciam-platformadmin", CspType: "aws"})
	assert.True(t, IsCspRoleProtectedError(err), "%v", err)
	assert.Nil(t, protectedRole, "보호 대상은 레코드를 만들지 않는다")
	assert.Zero(t, api.getCalls, "보호 대상은 AWS 를 조회하지 않는다")
	assert.Zero(t, countCspRoles(t, svc))

	// 비활성 Config 의 관리용 Role 도 보호한다
	seedIdpConfig(t, svc.db, "aws-old", "aws", model.AuthMethodOIDC, false, map[string]string{"role_arn": "arn:aws:iam::111111111111:role/mciam-admin"})
	_, _, err = svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "admin", CspType: "aws", CspIdpConfigID: nil})
	assert.True(t, IsCspRoleProtectedError(err), "%v", err)
}

func TestResolveTargetCspRole_GetRoleErrorDoesNotCreate(t *testing.T) {
	svc, api, _ := newTargetTestService(t, nil)
	api.getErr = errors.New("AccessDenied")
	_, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-x", CspType: "aws"})
	require.Error(t, err)
	assert.False(t, IsCspRoleRequestError(err))
	assert.Empty(t, api.createCalls, "NoSuchEntity 가 아니면 생성하지 않는다")
	assert.Zero(t, countCspRoles(t, svc))
}

func TestResolveTargetCspRole_InvalidInputsWriteNothing(t *testing.T) {
	svc, api, _ := newTargetTestService(t, map[string]string{"role_arn": testManagementRoleArn, "oidc_provider_arn": testOidcProviderArn})
	invalidRole, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-noaud", CspType: "aws"})
	assert.True(t, IsCspRoleRequestError(err), "Config audience 없음: %v", err)
	assert.Nil(t, invalidRole, "요청 오류는 레코드를 만들지 않는다")

	_, _, err = svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-" + string(make([]byte, 60)), CspType: "aws"})
	assert.True(t, IsCspRoleRequestError(err), "이름 규칙 위반: %v", err)

	_, _, err = svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "", CspType: "aws"})
	assert.True(t, IsCspRoleRequestError(err), "빈 이름: %v", err)

	assert.Empty(t, api.createCalls)
	assert.Zero(t, countCspRoles(t, svc))
}

func TestResolveTargetCspRole_FailedRecordKeptAndReused(t *testing.T) {
	svc, api, _ := newTargetTestService(t, nil)
	api.createErr = errors.New("MalformedPolicyDocument")

	returned, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-fail", CspType: "aws"})
	require.Error(t, err)
	assert.False(t, IsCspRoleRequestError(err) || IsCspRoleProtectedError(err), "CreateRole 실패는 csp_error 성격")
	failed, err := svc.cspRoleRepo.GetCspRoleByName("mciam-t045-fail", "aws")
	require.NoError(t, err)
	require.NotNil(t, failed, "실패해도 레코드는 남긴다")
	assert.Equal(t, "failed", failed.Status)
	require.NotNil(t, returned, "CreateRole 실패 시 failed 레코드를 돌려준다")
	assert.Equal(t, failed.ID, returned.ID)

	// 같은 이름으로 다시 실패해도 같은 failed 레코드를 이어받는다
	again, _, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-fail", CspType: "aws"})
	require.Error(t, err)
	require.NotNil(t, again)
	assert.Equal(t, failed.ID, again.ID)

	// 사용자가 콘솔에서 Role 을 만든 뒤 다시 저장하면 같은 레코드로 이어진다
	api.createErr = nil
	api.roles["mciam-t045-fail"] = fakeAwsRole("mciam-t045-fail", `{"Statement":[{"Principal":{"Federated":"`+testOidcProviderArn+`"}}]}`)
	role, result, err := svc.ResolveTargetCspRole(&model.CreateCspRoleRequest{CspRoleName: "t045-fail", CspType: "aws"})
	require.NoError(t, err)
	assert.Equal(t, TargetResultRegisteredAndMapped, result)
	assert.Equal(t, failed.ID, role.ID)
	assert.Equal(t, "created", role.Status)
	assert.Equal(t, int64(1), countCspRoles(t, svc))
}

func TestCheckTargetCspRole_NoWrites(t *testing.T) {
	svc, api, _ := newTargetTestService(t, nil)
	api.roles["mciam-t045-pre"] = fakeAwsRole("mciam-t045-pre", "{}")
	registered := &model.CspRole{Name: "mciam-t045-db", CspType: "aws", Status: "created", IamIdentifier: "arn:aws:iam::111111111111:role/mciam-t045-db"}
	require.NoError(t, svc.db.Create(registered).Error)
	before := countCspRoles(t, svc)

	cases := map[string]string{
		"t045-db":             TargetCheckExistsInDB,
		"t045-pre":            TargetCheckExistsInCsp,
		"t045-new":            TargetCheckNotFound,
		"mciam-platformadmin": TargetCheckProtected,
		"":                    TargetCheckInvalid,
	}
	for name, want := range cases {
		got := svc.CheckTargetCspRole(model.CreateCspRoleRequest{CspRoleName: name, CspType: "aws", AuthMethod: constants.AuthMethodOIDC})
		assert.Equal(t, want, got.Status, "%q: %+v", name, got)
	}
	assert.Equal(t, before, countCspRoles(t, svc), "check 는 DB 에 쓰지 않는다")
	assert.Empty(t, api.createCalls, "check 는 AWS 에 쓰지 않는다")
}
