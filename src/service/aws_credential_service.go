package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/m-cmp/mc-iam-manager/model"
)

// AwsCredentialService defines operations for interacting with AWS STS and IAM.
type AwsCredentialService interface {
	AssumeRoleWithWebIdentity(ctx context.Context, roleArn, kcUserId, webIdentityToken, idpArn, region string) (*model.CspCredentialResponse, error)
	AssumeRoleWithSAML(ctx context.Context, roleArn, principalArn, samlAssertion, region string) (*model.CspCredentialResponse, error)
	// CheckOIDCProvider AWS IAM OIDC Provider 존재 및 audience 확인
	CheckOIDCProvider(ctx context.Context, oidcProviderArn string) (string, error)
	// CheckSAMLProvider AWS IAM SAML Provider 존재 확인
	CheckSAMLProvider(ctx context.Context, samlProviderArn string) (string, error)
	// CheckRoleTrust AWS IAM Role Trust Policy 확인
	CheckRoleTrust(ctx context.Context, roleArn, expectedAction, expectedProviderArn string) (string, error)
	// CheckCallerIdentity SECRET_KEY 자격증명 유효성 확인 (STS GetCallerIdentity)
	CheckCallerIdentity(ctx context.Context, accessKeyID, secretKey string) (string, error)
}

// awsCredentialService implements AwsCredentialService.
type awsCredentialService struct {
	// No fields needed for now, AWS config loaded dynamically
}

// NewAwsCredentialService creates a new AwsCredentialService.
func NewAwsCredentialService() AwsCredentialService {
	return &awsCredentialService{}
}

// AssumeRoleWithWebIdentity assumes an IAM role using a web identity token (OIDC).
// kcUserId is used to generate a unique RoleSessionName.
func (s *awsCredentialService) AssumeRoleWithWebIdentity(ctx context.Context, roleArn, kcUserId, webIdentityToken, idpArn, region string) (*model.CspCredentialResponse, error) {
	// Load default AWS configuration
	log.Printf("[AWS_CREDENTIAL] Loading AWS configuration...")
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("[AWS_CREDENTIAL] Unable to load AWS SDK config: %v", err)

		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Set region if provided
	if region != "" {
		awsCfg.Region = region
	} else if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
		awsCfg.Region = envRegion
	}
	log.Printf("[AWS_CREDENTIAL] Using AWS Region: %s for STS call", awsCfg.Region)

	stsClient := sts.NewFromConfig(awsCfg)

	// Create a unique RoleSessionName, e.g., using user ID and timestamp
	// Must be between 2 and 64 characters.
	roleSessionName := fmt.Sprintf("mciam-%s-%d", kcUserId, time.Now().Unix())
	if len(roleSessionName) > 64 {
		roleSessionName = roleSessionName[:64] // Truncate if too long
	}

	input := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          &roleArn,
		RoleSessionName:  &roleSessionName,
		WebIdentityToken: &webIdentityToken,
		DurationSeconds:  nil, // Use default duration (1 hour)
	}

	log.Printf("[AWS_CREDENTIAL] Attempting to assume role %s with web identity token for session %s", roleArn, roleSessionName)
	result, err := stsClient.AssumeRoleWithWebIdentity(ctx, input)
	if err != nil {
		log.Printf("[AWS_CREDENTIAL] AWS AssumeRoleWithWebIdentity failed for role %s: %v", roleArn, err)
		return nil, fmt.Errorf("failed to assume AWS role %s: %w", roleArn, err)
	}

	if result.Credentials == nil {
		return nil, fmt.Errorf("received nil credentials from AWS STS for role %s", roleArn)
	}

	log.Printf("[AWS_CREDENTIAL] Successfully assumed role %s, Expiration: %s", roleArn, result.Credentials.Expiration.String())

	// Map STS response to our generic response model
	response := &model.CspCredentialResponse{
		CspType:         "aws",
		AccessKeyId:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          awsCfg.Region,
	}

	return response, nil
}

// CheckOIDCProvider AWS IAM OIDC Provider 존재 및 audience 확인
func (s *awsCredentialService) CheckOIDCProvider(ctx context.Context, oidcProviderArn string) (string, error) {
	cfg, err := newAWSIAMConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("IAM 읽기 권한 없음 (degraded mode) — %v", err)
	}
	iamClient := iam.NewFromConfig(cfg)
	result, err := iamClient.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: &oidcProviderArn,
	})
	if err != nil {
		return "", fmt.Errorf("OIDC Provider 없음: %v — AWS IAM에 Keycloak issuer URL로 OIDC Provider 생성 필요", err)
	}
	audiences := make([]string, len(result.ClientIDList))
	copy(audiences, result.ClientIDList)
	return fmt.Sprintf("OIDC Provider 존재 확인, audiences=%v", audiences), nil
}

// CheckSAMLProvider AWS IAM SAML Provider 존재 확인
func (s *awsCredentialService) CheckSAMLProvider(ctx context.Context, samlProviderArn string) (string, error) {
	cfg, err := newAWSIAMConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("IAM 읽기 권한 없음 (degraded mode) — %v", err)
	}
	iamClient := iam.NewFromConfig(cfg)
	_, err = iamClient.GetSAMLProvider(ctx, &iam.GetSAMLProviderInput{
		SAMLProviderArn: &samlProviderArn,
	})
	if err != nil {
		return "", fmt.Errorf("SAML Provider 없음: %v — AWS IAM에 Keycloak 메타데이터로 SAML Provider 생성 필요", err)
	}
	return fmt.Sprintf("SAML Provider 존재 확인: %s", samlProviderArn), nil
}

// CheckRoleTrust AWS IAM Role Trust Policy 확인
func (s *awsCredentialService) CheckRoleTrust(ctx context.Context, roleArn, requiredAction, requiredPrincipal string) (string, error) {
	cfg, err := newAWSIAMConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("IAM 읽기 권한 없음 (degraded mode) — %v", err)
	}

	// ARN에서 role name 추출 (arn:aws:iam::ACCOUNT:role/ROLE_NAME)
	parts := strings.Split(roleArn, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("roleArn 형식 오류: %s", roleArn)
	}
	roleName := parts[len(parts)-1]

	iamClient := iam.NewFromConfig(cfg)
	result, err := iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: &roleName,
	})
	if err != nil {
		return "", fmt.Errorf("IAM Role 조회 실패: %v — Role ARN 확인 필요", err)
	}

	trustDoc := ""
	if result.Role.AssumeRolePolicyDocument != nil {
		trustDoc = *result.Role.AssumeRolePolicyDocument
	}

	if !strings.Contains(trustDoc, requiredAction) {
		return "", fmt.Errorf("Trust Policy에 %s 없음 — IAM Role Trust Relationship에 %s 추가 필요", requiredAction, requiredAction)
	}
	return fmt.Sprintf("Trust Policy에 %s 확인 완료", requiredAction), nil
}

// CheckCallerIdentity SECRET_KEY 자격증명 유효성 확인 (STS GetCallerIdentity SDK signed call)
func (s *awsCredentialService) CheckCallerIdentity(ctx context.Context, accessKeyID, secretKey string) (string, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-northeast-2"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
		),
	)
	if err != nil {
		return "", fmt.Errorf("AWS 설정 로드 실패: %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("AWS STS GetCallerIdentity 실패 — access_key_id/secret_access_key 유효하지 않음: %v", err)
	}
	return fmt.Sprintf("AWS 자격증명 확인 완료 — Account=%s Arn=%s", aws.ToString(result.Account), aws.ToString(result.Arn)), nil
}

// newAWSIAMConfig IAM 읽기용 AWS 설정 로드 — 자격증명 없으면 오류 반환
// IAM은 global service이지만 SDK는 region 필요 — AWS_REGION 또는 기본값 us-east-1 사용
func newAWSIAMConfig(ctx context.Context) (aws.Config, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return aws.Config{}, fmt.Errorf("AWS 자격증명 없음: %v", err)
	}
	// 자격증명이 실제로 있는지 확인
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil || creds.AccessKeyID == "" {
		return aws.Config{}, fmt.Errorf("AWS 자격증명 없음 — IAM 읽기 권한 설정 필요")
	}
	return cfg, nil
}

// AssumeRoleWithSAML assumes an IAM role using a SAML assertion.
// principalArn is the SAML provider ARN (e.g., arn:aws:iam::ACCOUNT:saml-provider/NAME).
// roleArn is the IAM role ARN to assume.
// samlAssertion is the base64-encoded SAML assertion from the IdP.
func (s *awsCredentialService) AssumeRoleWithSAML(ctx context.Context, roleArn, principalArn, samlAssertion, region string) (*model.CspCredentialResponse, error) {
	log.Printf("[AWS_CREDENTIAL] Loading AWS configuration for SAML...")
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("[AWS_CREDENTIAL] Unable to load AWS SDK config: %v", err)
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	if region != "" {
		awsCfg.Region = region
	} else if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
		awsCfg.Region = envRegion
	}
	log.Printf("[AWS_CREDENTIAL] Using AWS Region: %s for SAML STS call", awsCfg.Region)

	stsClient := sts.NewFromConfig(awsCfg)

	input := &sts.AssumeRoleWithSAMLInput{
		RoleArn:      &roleArn,
		PrincipalArn: &principalArn,
		SAMLAssertion: &samlAssertion,
	}

	log.Printf("[AWS_CREDENTIAL] Attempting AssumeRoleWithSAML for role %s with principal %s", roleArn, principalArn)
	result, err := stsClient.AssumeRoleWithSAML(ctx, input)
	if err != nil {
		log.Printf("[AWS_CREDENTIAL] AWS AssumeRoleWithSAML failed for role %s: %v", roleArn, err)
		return nil, fmt.Errorf("failed to assume AWS role via SAML %s: %w", roleArn, err)
	}

	if result.Credentials == nil {
		return nil, fmt.Errorf("received nil credentials from AWS STS (SAML) for role %s", roleArn)
	}

	log.Printf("[AWS_CREDENTIAL] Successfully assumed role via SAML %s, Expiration: %s", roleArn, result.Credentials.Expiration.String())

	return &model.CspCredentialResponse{
		CspType:         "aws",
		AccessKeyId:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          awsCfg.Region,
	}, nil
}
