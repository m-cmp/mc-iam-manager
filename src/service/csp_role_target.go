package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
)

// 여정 1: Master Role 저장 시 대상 CSP Role 을 확정한다 (IAM-BUG-043, IAM-TECH-045 M1).
// 판단 순서: 이름 정규화(mciam-) → 관리용 Role 보호 → 소속 IdP Config → DB → AWS GetRole → (없을 때만) CreateRole → 매핑 검증

// 대상 CSP Role 처리 결과 (cspRoleResults[].result)
const (
	TargetResultMapped              = "mapped"                // DB 에 있던 CspRole 에 매핑
	TargetResultRegisteredAndMapped = "registered_and_mapped" // AWS 에 있던 Role 을 등록하고 매핑
	TargetResultCreatedAndMapped    = "created_and_mapped"    // AWS 에 Role 을 만들고 매핑
	TargetResultUnchanged           = "unchanged"             // (PUT) 이미 매핑돼 있어 변경 없음
	TargetResultUnmapped            = "unmapped"              // (PUT) 요청에서 빠져 매핑 해제
	TargetResultFailed              = "failed"                // 이 항목의 매핑만 빠짐
)

// 존재 확인 결과 (POST /roles/csp/check)
const (
	TargetCheckExistsInDB  = "exists_in_db"  // IAM 에 등록돼 있음 → 매핑만
	TargetCheckExistsInCsp = "exists_in_csp" // AWS 에만 있음 → 저장 시 등록 후 매핑
	TargetCheckNotFound    = "not_found"     // 어디에도 없음 → 저장 시 생성 후 매핑
	TargetCheckProtected   = "protected"     // 관리용 Role → 매핑 불가
	TargetCheckInvalid     = "invalid"       // 요청 값 오류 (이름, 소속 Config 등)
	TargetCheckError       = "error"         // 조회 실패 (AWS 오류 등)
)

// CspRoleProtectedError 관리용 Role(IdP Config 의 role_arn)을 매핑·변경하려는 경우 (핸들러는 409)
type CspRoleProtectedError struct {
	Msg string
}

func (e *CspRoleProtectedError) Error() string { return e.Msg }

// IsCspRoleProtectedError err 가 CspRoleProtectedError 인지 확인한다.
func IsCspRoleProtectedError(err error) bool {
	var e *CspRoleProtectedError
	return errors.As(err, &e)
}

// awsRoleAPI 대상 CSP Role 확정에 쓰는 AWS IAM 호출. *iam.Client 가 만족한다 (테스트는 fake).
type awsRoleAPI interface {
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
}

// awsRolePollInterval CreateRole 후 GetRole 재확인 간격 (테스트에서 줄인다)
var awsRolePollInterval = time.Second

// AWS IAM Role 이름 규칙: 1~64자, 영숫자와 +=,.@_-
var awsRoleNamePattern = regexp.MustCompile(`^[\w+=,.@-]{1,64}$`)

// NormalizeTargetCspRoleName 요청 cspRoleName 을 대상 CSP Role 이름으로 정규화한다.
// 비어 있으면 요청 오류(자동 명명하지 않음, IAM-BUG-043). mciam- 접두사가 없으면 붙인다.
func NormalizeTargetCspRoleName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", cspRoleRequestErrorf("cspRoleName is required")
	}
	if !strings.HasPrefix(name, constants.CspRoleNamePrefix) {
		name = constants.CspRoleNamePrefix + name
	}
	return name, nil
}

func validateTargetCspRoleName(name, cspType string) error {
	if cspType == "aws" && !awsRoleNamePattern.MatchString(name) {
		return cspRoleRequestErrorf("invalid AWS role name %q: 1-64 characters of letters, digits and +=,.@_-", name)
	}
	return nil
}

// roleNameFromArn arn:aws:iam::<account>:role/<path/>name → name
func roleNameFromArn(arn string) string {
	i := strings.LastIndex(arn, "/")
	if i < 0 {
		return ""
	}
	return arn[i+1:]
}

// buildAwsOidcTrustPolicy 소속 IdP Config 의 oidc_provider_arn·audience 로 신뢰 정책을 만든다 (IAM-TECH-045 M1, env 사용 안 함).
func buildAwsOidcTrustPolicy(cfg *model.CspIdpConfig) (string, error) {
	if cfg == nil {
		return "", cspRoleRequestErrorf("IdP config is required to build the trust policy")
	}
	providerArn := cfg.GetOidcProviderArn()
	const marker = ":oidc-provider/"
	i := strings.Index(providerArn, marker)
	if !strings.HasPrefix(providerArn, "arn:aws:iam::") || i < 0 || len(providerArn) == i+len(marker) {
		return "", cspRoleRequestErrorf("IdP config %d (%s): oidc_provider_arn %q is not an AWS OIDC provider ARN", cfg.ID, cfg.Name, providerArn)
	}
	audience := cfg.GetAudience()
	if audience == "" || audience == "notyet" {
		return "", cspRoleRequestErrorf("IdP config %d (%s): audience is not set", cfg.ID, cfg.Name)
	}
	host := providerArn[i+len(marker):]
	doc := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{{
			"Effect":    "Allow",
			"Principal": map[string]string{"Federated": providerArn},
			"Action":    "sts:AssumeRoleWithWebIdentity",
			"Condition": map[string]interface{}{
				"StringEquals": map[string]string{host + ":aud": audience},
			},
		}},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// buildAwsTrustPolicy 인증 방식에 맞는 신뢰 정책 문서와 Federated principal
func buildAwsTrustPolicy(req *model.CreateCspRoleRequest, cfg *model.CspIdpConfig) (string, string, error) {
	if requestAuthMethod(req) == constants.AuthMethodSAML {
		if req.IdpIdentifier == "" && cfg != nil {
			req.IdpIdentifier = cfg.GetSamlProviderArn()
		}
		doc, err := getAwsSamlAssumeRolePolicyDocument(req)
		return doc, req.IdpIdentifier, err
	}
	doc, err := buildAwsOidcTrustPolicy(cfg)
	if err != nil {
		return "", "", err
	}
	return doc, cfg.GetOidcProviderArn(), nil
}

// isAwsAccessDenied 관리용 Role 에 해당 IAM 권한이 없어 거부된 오류인지
func isAwsAccessDenied(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "AccessDenied", "AccessDeniedException", "UnauthorizedOperation":
			return true
		}
	}
	return false
}

// verifyAwsOidcProvider 소속 Config 의 oidc_provider_arn 이 AWS 에 있고, audience 가 provider 의 ClientIDList 에 있는지 확인한다.
//   - provider 가 없거나 audience 가 등록돼 있지 않으면 요청 오류(그 항목만 failed). 그대로 만들면 발급이 항상 실패한다
//   - 관리용 Role 에 iam:GetOpenIDConnectProvider 권한이 없으면 WARN 을 남기고 건너뛴다(하위호환)
func verifyAwsOidcProvider(ctx context.Context, api awsRoleAPI, cfg *model.CspIdpConfig) error {
	if cfg == nil {
		return nil
	}
	providerArn := cfg.GetOidcProviderArn()
	out, err := api.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{OpenIDConnectProviderArn: aws.String(providerArn)})
	switch {
	case err == nil:
	case isAwsNoSuchEntity(err):
		return cspRoleRequestErrorf("IdP config %d (%s): OIDC provider %s does not exist in AWS", cfg.ID, cfg.Name, providerArn)
	case isAwsAccessDenied(err):
		log.Printf("[WARN] IdP config %d: cannot verify OIDC provider %s (iam:GetOpenIDConnectProvider denied) — skipped: %v", cfg.ID, providerArn, err)
		return nil
	default:
		return fmt.Errorf("failed to check OIDC provider %s: %w", providerArn, err)
	}
	audience := cfg.GetAudience()
	if out != nil && audience != "" && !containsString(out.ClientIDList, audience) {
		return cspRoleRequestErrorf("IdP config %d (%s): audience %q is not in the client ID list of OIDC provider %s (%s)",
			cfg.ID, cfg.Name, audience, providerArn, strings.Join(out.ClientIDList, ", "))
	}
	return nil
}

func isAwsNoSuchEntity(err error) bool {
	var nse *iamtypes.NoSuchEntityException
	return errors.As(err, &nse)
}

// awsRoleClientFor 소속 Config 기준 관리용 AWS IAM 클라이언트. 테스트는 awsRoleClient 를 바꿔 끼운다.
func (s *CspRoleService) awsRoleClientFor(cfg *model.CspIdpConfig) (awsRoleAPI, error) {
	if s.awsRoleClient != nil {
		return s.awsRoleClient(cfg)
	}
	issuedBy := "system"
	credential, err := s.tempCredentialRepo.GetOrCreateValidCredential("aws", "oidc", "ap-northeast-2", nil, issuedBy, func() (*model.TempCredential, error) {
		return s.createNewAwsCredential(issuedBy)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create valid credential: %v", err)
	}
	awsCfg, err := s.createAwsConfigWithTempCredential(credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config with temp credential: %v", err)
	}
	return iam.NewFromConfig(awsCfg), nil
}

// ensureAwsRole AWS 에 Role 이 있으면 등록, NoSuchEntity 일 때만 소속 Config 기반 신뢰 정책으로 생성한다.
// GetRole 이 다른 이유로 실패하면 생성하지 않는다. CreateRole 이 실패하면 failed 레코드를 남긴다(다시 저장하면 이어받음).
func (s *CspRoleService) ensureAwsRole(api awsRoleAPI, req *model.CreateCspRoleRequest, idpConfig *model.CspIdpConfig) (*model.CspRole, string, error) {
	ctx := context.TODO()
	got, getErr := api.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(req.CspRoleName)})
	if getErr != nil && !isAwsNoSuchEntity(getErr) {
		return nil, "", fmt.Errorf("failed to check AWS role %s: %w", req.CspRoleName, getErr)
	}

	// 신뢰 정책은 레코드를 만들기 전에 검증한다 (Config 값이 틀리면 아무것도 쓰지 않는다)
	var trustDoc, federated string
	if getErr != nil {
		var err error
		if trustDoc, federated, err = buildAwsTrustPolicy(req, idpConfig); err != nil {
			return nil, "", err
		}
		// AWS 는 없는 provider 를 가리키는 신뢰 정책도 받아들이므로 만들기 전에 확인한다 (IAM-TECH-050)
		if requestAuthMethod(req) != constants.AuthMethodSAML {
			if err := verifyAwsOidcProvider(ctx, api, idpConfig); err != nil {
				return nil, "", err
			}
		}
	}

	// failed 레코드는 이어받아 저장된 값을 지우지 않는다
	existing, err := s.cspRoleRepo.GetCspRoleByName(req.CspRoleName, req.CspType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing CSP role: %w", err)
	}
	role := &model.CspRole{}
	if existing != nil {
		role = existing
	}
	role.Name = req.CspRoleName
	role.CspType = req.CspType
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.IdpIdentifier != "" {
		role.IdpIdentifier = req.IdpIdentifier
	}
	if req.IamIdentifier != "" {
		role.IamIdentifier = req.IamIdentifier
	}
	if len(req.ExtendedConfig) > 0 {
		role.ExtendedConfig = req.ExtendedConfig
	}
	applyCspIdpConfig(role, idpConfig)

	if getErr == nil {
		role.IdpIdentifier = resolveIdpIdentifier(role.Name, req.IdpIdentifier, idpConfig,
			federatedFromTrustPolicy(aws.ToString(got.Role.AssumeRolePolicyDocument)))
		if role.ID == 0 {
			role.Status = "created"
			if err := s.cspRoleRepo.CreateCspRoleRecord(role); err != nil {
				return nil, "", fmt.Errorf("failed to create CSP role record: %w", err)
			}
		}
		saved, err := s.syncAwsRoleToDb(role, got)
		return saved, TargetResultRegisteredAndMapped, err
	}

	role.Status = "creating"
	if role.ID == 0 {
		err = s.cspRoleRepo.CreateCspRoleRecord(role)
	} else {
		err = s.cspRoleRepo.UpdateCspRoleRecord(role)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to save CSP role record: %w", err)
	}
	fail := func(cause error) (*model.CspRole, string, error) {
		role.Status = "failed"
		if uerr := s.cspRoleRepo.UpdateCspRoleRecord(role); uerr != nil {
			log.Printf("[WARN] CSP role %s: failed to mark record as failed: %v", role.Name, uerr)
		}
		// failed 레코드를 함께 돌려준다 (응답 cspRoleResults[].cspRoleId 로 알린다)
		return role, "", cause
	}

	if _, err := api.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(req.CspRoleName),
		AssumeRolePolicyDocument: aws.String(trustDoc),
		Description:              aws.String(role.Description),
	}); err != nil {
		return fail(fmt.Errorf("failed to create AWS role %s: %w", req.CspRoleName, err))
	}
	for i := 0; i < 30; i++ {
		time.Sleep(awsRolePollInterval)
		res, err := api.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(req.CspRoleName)})
		if err == nil && res != nil && res.Role != nil {
			applyAwsRoleToModel(role, federated, res)
			if err := s.cspRoleRepo.UpdateCspRoleRecord(role); err != nil {
				return nil, "", fmt.Errorf("failed to update CSP role in database: %v", err)
			}
			return role, TargetResultCreatedAndMapped, nil
		}
	}
	return fail(fmt.Errorf("failed to verify AWS role %s creation after 30 attempts", req.CspRoleName))
}

// ResolveTargetCspRole 대상 CSP Role 을 확정하고(없으면 생성) 매핑 가능한 상태인지 검증한다.
// 돌려주는 result 는 TargetResult* 중 하나. 오류는 CspRoleRequestError(400 성격)·CspRoleProtectedError(409 성격)·그 외로 구분된다.
// 오류여도 레코드가 남았으면(CreateRole 실패로 남긴 failed 레코드, 매핑 검증에 걸린 기존 레코드) 그 레코드를 함께 돌려준다.
func (s *CspRoleService) ResolveTargetCspRole(req *model.CreateCspRoleRequest) (*model.CspRole, string, error) {
	name, err := NormalizeTargetCspRoleName(req.CspRoleName)
	if err != nil {
		return nil, "", err
	}
	req.CspRoleName = name
	if req.CspType == "" {
		return nil, "", cspRoleRequestErrorf("cspType is required for CSP role %s", name)
	}
	if err := validateTargetCspRoleName(name, req.CspType); err != nil {
		return nil, "", err
	}

	if err := s.GuardNotManagementRole(name, "", "mapped"); err != nil {
		return nil, "", err
	}

	idpConfig, err := s.resolveCspIdpConfig(req)
	if err != nil {
		return nil, "", err
	}

	existing, err := s.cspRoleRepo.GetCspRoleByName(name, req.CspType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing CSP role: %w", err)
	}

	var role *model.CspRole
	result := TargetResultMapped
	switch {
	case existing != nil && existing.Status != "failed":
		role = existing
	case req.CspType == "aws":
		api, err := s.awsRoleClientFor(idpConfig)
		if err != nil {
			return nil, "", err
		}
		if role, result, err = s.ensureAwsRole(api, req, idpConfig); err != nil {
			return role, "", err // CreateRole 실패면 failed 레코드, 그 외는 nil
		}
	default:
		// AWS 이외: 클라우드 쪽 생성 없이 DB 등록만 (기존 동작)
		if role, err = s.CreateCspRole(req); err != nil {
			return nil, "", err
		}
		result = TargetResultRegisteredAndMapped
	}

	if err := s.ValidateCspRoleForMapping(role.ID, requestAuthMethod(req), req.CspType); err != nil {
		return role, "", err
	}
	return role, result, nil
}

// CspRoleCheckResult 존재 확인 결과 한 건
type CspRoleCheckResult struct {
	CspRoleName string `json:"cspRoleName"`
	CspType     string `json:"cspType"`
	Status      string `json:"status"`
	CspRoleID   uint   `json:"cspRoleId,omitempty"`
	Arn         string `json:"arn,omitempty"`
	Message     string `json:"message,omitempty"`
}

// CheckTargetCspRole 대상 CSP Role 이 어디에 있는지만 확인한다. DB·AWS 어디에도 쓰지 않는다.
func (s *CspRoleService) CheckTargetCspRole(req model.CreateCspRoleRequest) CspRoleCheckResult {
	out := CspRoleCheckResult{CspRoleName: req.CspRoleName, CspType: req.CspType}
	name, err := NormalizeTargetCspRoleName(req.CspRoleName)
	if err == nil && req.CspType == "" {
		err = cspRoleRequestErrorf("cspType is required")
	}
	if err == nil {
		err = validateTargetCspRoleName(name, req.CspType)
	}
	if err != nil {
		out.Status, out.Message = TargetCheckInvalid, err.Error()
		return out
	}
	out.CspRoleName = name
	req.CspRoleName = name

	if protected, cfg, err := s.isManagementRole(name, ""); err != nil {
		out.Status, out.Message = TargetCheckError, err.Error()
		return out
	} else if protected {
		out.Status = TargetCheckProtected
		out.Message = fmt.Sprintf("management role of IdP config %d (%s)", cfg.ID, cfg.Name)
		return out
	}

	idpConfig, err := s.resolveCspIdpConfig(&req)
	if err != nil {
		out.Status, out.Message = TargetCheckInvalid, err.Error()
		return out
	}

	existing, err := s.cspRoleRepo.GetCspRoleByName(name, req.CspType)
	if err != nil {
		out.Status, out.Message = TargetCheckError, err.Error()
		return out
	}
	if existing != nil && existing.Status != "failed" {
		out.Status, out.CspRoleID, out.Arn = TargetCheckExistsInDB, existing.ID, existing.IamIdentifier
		return out
	}
	if req.CspType != "aws" {
		out.Status = TargetCheckNotFound
		return out
	}

	api, err := s.awsRoleClientFor(idpConfig)
	if err != nil {
		out.Status, out.Message = TargetCheckError, err.Error()
		return out
	}
	got, err := api.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String(name)})
	switch {
	case err == nil:
		out.Status, out.Arn = TargetCheckExistsInCsp, aws.ToString(got.Role.Arn)
	case isAwsNoSuchEntity(err):
		out.Status = TargetCheckNotFound
		if _, _, terr := buildAwsTrustPolicy(&req, idpConfig); terr != nil {
			out.Message = "the role cannot be created: " + terr.Error()
		} else if requestAuthMethod(&req) != constants.AuthMethodSAML {
			if perr := verifyAwsOidcProvider(context.TODO(), api, idpConfig); perr != nil {
				out.Message = "the role cannot be created: " + perr.Error()
			}
		}
	default:
		out.Status, out.Message = TargetCheckError, err.Error()
	}
	return out
}
