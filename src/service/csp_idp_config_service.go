package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CspIdpConfigService CSP IDP 설정 서비스
type CspIdpConfigService struct {
	db               *gorm.DB
	cspIdpConfigRepo *repository.CspIdpConfigRepository
	cspAccountRepo   *repository.CspAccountRepository
	keycloakService  KeycloakService
}

// NewCspIdpConfigService 새 CspIdpConfigService 인스턴스 생성
func NewCspIdpConfigService(db *gorm.DB, keycloakService KeycloakService) *CspIdpConfigService {
	return &CspIdpConfigService{
		db:               db,
		cspIdpConfigRepo: repository.NewCspIdpConfigRepository(db),
		cspAccountRepo:   repository.NewCspAccountRepository(db),
		keycloakService:  keycloakService,
	}
}

// CreateCspIdpConfig CSP IDP 설정 생성
func (s *CspIdpConfigService) CreateCspIdpConfig(req *model.CreateCspIdpConfigRequest) (*model.CspIdpConfig, error) {
	// CSP 계정 존재 확인
	account, err := s.cspAccountRepo.GetByID(req.CspAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("CSP account not found with ID: %d", req.CspAccountID)
	}

	// 이름 중복 확인
	exists, err := s.cspIdpConfigRepo.ExistsByNameAndAccountID(req.Name, req.CspAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check IDP config existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("IDP config with name '%s' already exists for this account", req.Name)
	}

	// IDP 설정 생성
	idpConfig := &model.CspIdpConfig{
		Name:         req.Name,
		CspAccountID: req.CspAccountID,
		AuthMethod:   req.AuthMethod,
		Config:       req.Config,
		IsActive:     true,
		Description:  req.Description,
	}

	if err := s.cspIdpConfigRepo.Create(idpConfig); err != nil {
		return nil, fmt.Errorf("failed to create IDP config: %w", err)
	}

	log.Printf("Created CSP IDP config: %s (method: %s)", idpConfig.Name, idpConfig.AuthMethod)
	return idpConfig, nil
}

// GetCspIdpConfigByID ID로 CSP IDP 설정 조회
func (s *CspIdpConfigService) GetCspIdpConfigByID(id uint) (*model.CspIdpConfig, error) {
	idpConfig, err := s.cspIdpConfigRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDP config: %w", err)
	}
	if idpConfig == nil {
		return nil, fmt.Errorf("IDP config not found with ID: %d", id)
	}
	return idpConfig, nil
}

// ListCspIdpConfigs CSP IDP 설정 목록 조회
func (s *CspIdpConfigService) ListCspIdpConfigs(filter *model.CspIdpConfigFilter) ([]*model.CspIdpConfig, error) {
	configs, err := s.cspIdpConfigRepo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list IDP configs: %w", err)
	}
	return configs, nil
}

// UpdateCspIdpConfig CSP IDP 설정 수정
func (s *CspIdpConfigService) UpdateCspIdpConfig(id uint, req *model.UpdateCspIdpConfigRequest) (*model.CspIdpConfig, error) {
	// 기존 설정 조회
	idpConfig, err := s.cspIdpConfigRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDP config: %w", err)
	}
	if idpConfig == nil {
		return nil, fmt.Errorf("IDP config not found with ID: %d", id)
	}

	// 필드 업데이트
	if req.Name != "" {
		// 이름 변경 시 중복 확인
		if req.Name != idpConfig.Name {
			exists, err := s.cspIdpConfigRepo.ExistsByNameAndAccountID(req.Name, idpConfig.CspAccountID)
			if err != nil {
				return nil, fmt.Errorf("failed to check IDP config existence: %w", err)
			}
			if exists {
				return nil, fmt.Errorf("IDP config with name '%s' already exists", req.Name)
			}
		}
		idpConfig.Name = req.Name
	}
	if req.Config != nil {
		idpConfig.Config = req.Config
	}
	if req.IsActive != nil {
		idpConfig.IsActive = *req.IsActive
	}
	if req.Description != "" {
		idpConfig.Description = req.Description
	}

	if err := s.cspIdpConfigRepo.Update(idpConfig); err != nil {
		return nil, fmt.Errorf("failed to update IDP config: %w", err)
	}

	log.Printf("Updated CSP IDP config: %s (ID: %d)", idpConfig.Name, idpConfig.ID)
	return idpConfig, nil
}

// DeleteCspIdpConfig CSP IDP 설정 삭제
func (s *CspIdpConfigService) DeleteCspIdpConfig(id uint) error {
	// 설정 존재 확인
	exists, err := s.cspIdpConfigRepo.ExistsByID(id)
	if err != nil {
		return fmt.Errorf("failed to check IDP config existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("IDP config not found with ID: %d", id)
	}

	// TODO: 연관된 CspRole 확인 (CspRole.CspIdpConfigID 참조)

	if err := s.cspIdpConfigRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete IDP config: %w", err)
	}

	log.Printf("Deleted CSP IDP config with ID: %d", id)
	return nil
}

// TestConnection IDP 연결 테스트
func (s *CspIdpConfigService) TestConnection(ctx context.Context, id uint) error {
	idpConfig, err := s.cspIdpConfigRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get IDP config: %w", err)
	}
	if idpConfig == nil {
		return fmt.Errorf("IDP config not found with ID: %d", id)
	}

	// CSP 계정 정보 조회
	account, err := s.cspAccountRepo.GetByID(idpConfig.CspAccountID)
	if err != nil {
		return fmt.Errorf("failed to get CSP account: %w", err)
	}

	// 인증 방식에 따른 연결 테스트
	switch idpConfig.AuthMethod {
	case model.AuthMethodOIDC:
		return s.testOidcConnection(ctx, idpConfig, account)
	case model.AuthMethodSAML:
		return s.testSamlConnection(ctx, idpConfig, account)
	case model.AuthMethodSecretKey:
		return s.testSecretKeyConnection(ctx, idpConfig, account)
	default:
		return fmt.Errorf("unsupported auth method: %s", idpConfig.AuthMethod)
	}
}

// testOidcConnection OIDC 연결 테스트
func (s *CspIdpConfigService) testOidcConnection(ctx context.Context, idpConfig *model.CspIdpConfig, account *model.CspAccount) error {
	switch account.CspType {
	case "aws":
		return s.testAwsOidcConnection(ctx, idpConfig, account)
	case "gcp":
		// TODO: GCP Workload Identity Federation 테스트
		return fmt.Errorf("GCP OIDC connection test not implemented yet")
	case "azure":
		// TODO: Azure AD Workload Identity 테스트
		return fmt.Errorf("Azure OIDC connection test not implemented yet")
	default:
		return fmt.Errorf("unsupported CSP type: %s", account.CspType)
	}
}

// testAwsOidcConnection AWS OIDC 연결 테스트
func (s *CspIdpConfigService) testAwsOidcConnection(ctx context.Context, idpConfig *model.CspIdpConfig, account *model.CspAccount) error {
	// Keycloak에서 OIDC 토큰 획득
	token, err := s.keycloakService.GetClientCredentialsToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Keycloak token: %w", err)
	}

	// AWS STS 클라이언트 생성
	region := account.GetRegion()
	if region == "" {
		region = "ap-northeast-2"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	// Role ARN 구성 (환경 변수 또는 IDP Config에서)
	roleArn := idpConfig.Config["role_arn"]
	if roleArn == "" {
		roleArn = os.Getenv("IDENTITY_ROLE_ARN_AWS")
	}
	if roleArn == "" {
		return fmt.Errorf("role_arn is not configured")
	}

	// AssumeRoleWithWebIdentity 테스트
	input := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleArn),
		RoleSessionName:  aws.String("mciam-connection-test"),
		WebIdentityToken: aws.String(token.AccessToken),
		DurationSeconds:  aws.Int32(900), // 최소 15분
	}

	result, err := stsClient.AssumeRoleWithWebIdentity(ctx, input)
	if err != nil {
		return fmt.Errorf("AWS OIDC connection test failed: %w", err)
	}

	log.Printf("AWS OIDC connection test successful. AssumedRoleUser: %s", *result.AssumedRoleUser.Arn)
	return nil
}

// testSamlConnection SAML 연결 테스트
func (s *CspIdpConfigService) testSamlConnection(ctx context.Context, idpConfig *model.CspIdpConfig, account *model.CspAccount) error {
	// TODO: SAML 연결 테스트 구현
	return fmt.Errorf("SAML connection test not implemented yet")
}

// testSecretKeyConnection Secret Key 연결 테스트
func (s *CspIdpConfigService) testSecretKeyConnection(ctx context.Context, idpConfig *model.CspIdpConfig, account *model.CspAccount) error {
	switch account.CspType {
	case "aws":
		return s.testAwsSecretKeyConnection(ctx, idpConfig, account)
	case "gcp":
		// TODO: GCP 서비스 계정 키 테스트
		return fmt.Errorf("GCP Secret Key connection test not implemented yet")
	case "azure":
		// TODO: Azure 서비스 프린시펄 테스트
		return fmt.Errorf("Azure Secret Key connection test not implemented yet")
	default:
		return fmt.Errorf("unsupported CSP type: %s", account.CspType)
	}
}

// testAwsSecretKeyConnection AWS Secret Key 연결 테스트
func (s *CspIdpConfigService) testAwsSecretKeyConnection(ctx context.Context, idpConfig *model.CspIdpConfig, account *model.CspAccount) error {
	accessKeyID := idpConfig.GetAccessKeyID()
	secretAccessKey := idpConfig.GetSecretAccessKey()

	if accessKeyID == "" || secretAccessKey == "" {
		return fmt.Errorf("access_key_id or secret_access_key is not configured")
	}

	// 암호화된 경우 복호화 필요
	if idpConfig.IsEncrypted() {
		// TODO: 복호화 로직 구현
		return fmt.Errorf("encrypted secret key decryption not implemented yet")
	}

	// AWS 설정 생성
	region := account.GetRegion()
	if region == "" {
		region = "ap-northeast-2"
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// STS GetCallerIdentity로 자격 증명 테스트
	stsClient := sts.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("AWS Secret Key connection test failed: %w", err)
	}

	log.Printf("AWS Secret Key connection test successful. Account: %s, Arn: %s", *result.Account, *result.Arn)
	return nil
}

// GetActiveIdpConfigsByAccountID 특정 계정의 활성 IDP 설정 목록 조회
func (s *CspIdpConfigService) GetActiveIdpConfigsByAccountID(accountID uint) ([]*model.CspIdpConfig, error) {
	configs, err := s.cspIdpConfigRepo.GetActiveByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active IDP configs: %w", err)
	}
	return configs, nil
}

// GetIdpConfigsByAuthMethod 인증 방식별 IDP 설정 목록 조회
func (s *CspIdpConfigService) GetIdpConfigsByAuthMethod(authMethod model.AuthMethodType) ([]*model.CspIdpConfig, error) {
	configs, err := s.cspIdpConfigRepo.GetByAuthMethod(authMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDP configs by auth method: %w", err)
	}
	return configs, nil
}

// ActivateIdpConfig IDP 설정 활성화
func (s *CspIdpConfigService) ActivateIdpConfig(id uint) error {
	idpConfig, err := s.cspIdpConfigRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get IDP config: %w", err)
	}
	if idpConfig == nil {
		return fmt.Errorf("IDP config not found with ID: %d", id)
	}

	idpConfig.IsActive = true
	if err := s.cspIdpConfigRepo.Update(idpConfig); err != nil {
		return fmt.Errorf("failed to activate IDP config: %w", err)
	}

	log.Printf("Activated CSP IDP config: %s (ID: %d)", idpConfig.Name, idpConfig.ID)
	return nil
}

// DeactivateIdpConfig IDP 설정 비활성화
func (s *CspIdpConfigService) DeactivateIdpConfig(id uint) error {
	idpConfig, err := s.cspIdpConfigRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get IDP config: %w", err)
	}
	if idpConfig == nil {
		return fmt.Errorf("IDP config not found with ID: %d", id)
	}

	idpConfig.IsActive = false
	if err := s.cspIdpConfigRepo.Update(idpConfig); err != nil {
		return fmt.Errorf("failed to deactivate IDP config: %w", err)
	}

	log.Printf("Deactivated CSP IDP config: %s (ID: %d)", idpConfig.Name, idpConfig.ID)
	return nil
}

// AssumeRoleWithIdpConfig IDP 설정을 사용하여 임시 자격 증명 획득
func (s *CspIdpConfigService) AssumeRoleWithIdpConfig(ctx context.Context, idpConfigID uint, roleArn string, sessionName string, durationSeconds int32) (*model.TempCredential, error) {
	idpConfig, err := s.cspIdpConfigRepo.GetByID(idpConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDP config: %w", err)
	}
	if idpConfig == nil {
		return nil, fmt.Errorf("IDP config not found with ID: %d", idpConfigID)
	}

	if !idpConfig.IsActive {
		return nil, fmt.Errorf("IDP config is not active")
	}

	account, err := s.cspAccountRepo.GetByID(idpConfig.CspAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}

	if account.CspType != "aws" {
		return nil, fmt.Errorf("only AWS is supported for AssumeRole currently")
	}

	region := account.GetRegion()
	if region == "" {
		region = "ap-northeast-2"
	}

	switch idpConfig.AuthMethod {
	case model.AuthMethodOIDC:
		return s.assumeRoleWithOidc(ctx, idpConfig, roleArn, sessionName, durationSeconds, region)
	case model.AuthMethodSecretKey:
		return s.assumeRoleWithSecretKey(ctx, idpConfig, roleArn, sessionName, durationSeconds, region)
	default:
		return nil, fmt.Errorf("unsupported auth method for AssumeRole: %s", idpConfig.AuthMethod)
	}
}

// assumeRoleWithOidc OIDC를 사용하여 역할 인수
func (s *CspIdpConfigService) assumeRoleWithOidc(ctx context.Context, idpConfig *model.CspIdpConfig, roleArn string, sessionName string, durationSeconds int32, region string) (*model.TempCredential, error) {
	// Keycloak에서 OIDC 토큰 획득
	token, err := s.keycloakService.GetClientCredentialsToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak token: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	if durationSeconds < 900 {
		durationSeconds = 3600 // 기본 1시간
	}

	input := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleArn),
		RoleSessionName:  aws.String(sessionName),
		WebIdentityToken: aws.String(token.AccessToken),
		DurationSeconds:  aws.Int32(durationSeconds),
	}

	result, err := stsClient.AssumeRoleWithWebIdentity(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role with OIDC: %w", err)
	}

	return &model.TempCredential{
		Provider:        "aws",
		AuthType:        "oidc",
		AccessKeyId:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Region:          region,
		IssuedAt:        time.Now(),
		ExpiresAt:       *result.Credentials.Expiration,
		IsActive:        true,
	}, nil
}

// assumeRoleWithSecretKey Secret Key를 사용하여 역할 인수
func (s *CspIdpConfigService) assumeRoleWithSecretKey(ctx context.Context, idpConfig *model.CspIdpConfig, roleArn string, sessionName string, durationSeconds int32, region string) (*model.TempCredential, error) {
	accessKeyID := idpConfig.GetAccessKeyID()
	secretAccessKey := idpConfig.GetSecretAccessKey()

	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("access_key_id or secret_access_key is not configured")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	if durationSeconds < 900 {
		durationSeconds = 3600 // 기본 1시간
	}

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(sessionName),
		DurationSeconds: aws.Int32(durationSeconds),
	}

	result, err := stsClient.AssumeRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role with secret key: %w", err)
	}

	return &model.TempCredential{
		Provider:        "aws",
		AuthType:        "secret_key",
		AccessKeyId:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Region:          region,
		IssuedAt:        time.Now(),
		ExpiresAt:       *result.Credentials.Expiration,
		IsActive:        true,
	}, nil
}
