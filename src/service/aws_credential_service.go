package service

import (
	"context"
	"fmt"
	"log" // For converting uint to string if needed for RoleSessionName
	"os"
	"time" // For RoleSessionName timestamp

	awsconfig "github.com/aws/aws-sdk-go-v2/config" // Alias to avoid conflict with our config pkg
	"github.com/aws/aws-sdk-go-v2/service/sts"

	// "github.com/aws/aws-sdk-go-v2/service/sts/types" // Not explicitly needed for this call
	"github.com/m-cmp/mc-iam-manager/model"
)

// AwsCredentialService defines operations for interacting with AWS STS.
type AwsCredentialService interface {
	AssumeRoleWithWebIdentity(ctx context.Context, roleArn, kcUserId, webIdentityToken, idpArn, region string) (*model.CspCredentialResponse, error)
	AssumeRoleWithSAML(ctx context.Context, roleArn, principalArn, samlAssertion, region string) (*model.CspCredentialResponse, error)
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
