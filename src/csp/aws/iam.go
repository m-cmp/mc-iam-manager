package aws

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"encoding/base64"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/m-cmp/mc-iam-manager/csp"
	"gorm.io/gorm"
)

// AssumeRoleWithWebIdentityResponse AWS STS 응답 구조체
type AssumeRoleWithWebIdentityResponse struct {
	XMLName                         xml.Name `xml:"AssumeRoleWithWebIdentityResponse"`
	AssumeRoleWithWebIdentityResult struct {
		Credentials struct {
			AccessKeyId     string `xml:"AccessKeyId"`
			SecretAccessKey string `xml:"SecretAccessKey"`
			SessionToken    string `xml:"SessionToken"`
			Expiration      string `xml:"Expiration"`
		} `xml:"Credentials"`
	} `xml:"AssumeRoleWithWebIdentityResult"`
}

// AWSIAMClient AWS IAM 클라이언트
type AWSIAMClient struct {
	client *iam.Client
	config *csp.IAMClientConfig
	db     *gorm.DB
}

// NewAWSIAMClient 새로운 AWS IAM 클라이언트 생성
func NewAWSIAMClient(cfg *csp.IAMClientConfig, db *gorm.DB) (*AWSIAMClient, error) {
	// 워크스페이스 ID 추출
	workspaceId := extractWorkspaceId(cfg.WorkspaceTicket)
	if workspaceId == "" {
		return nil, fmt.Errorf("invalid workspace ticket")
	}

	// 워크스페이스 역할 조회
	// var workspaceRole struct {
	// 	ID uint `gorm:"column:id"`
	// }
	// if err := db.Table("workspace_roles").
	// 	Where("workspace_id = ?", workspaceId).
	// 	First(&workspaceRole).Error; err != nil {
	// 	return nil, fmt.Errorf("failed to get workspace role: %w", err)
	// }

	// // CSP 역할 매핑 조회
	var cspRoleMapping struct {
		RoleARN string
	}
	//cspRoleMapping.RoleARN = "arn:aws:iam::050864702683:role/mciam_viewer"
	cspRoleMapping.RoleARN = "arn:aws:iam::050864702683:role/mciam-csp-role-manager"
	// if err := db.Table("workspace_role_csp_role_mappings").
	// 	Where("workspace_role_id = ? AND csp_type = ?", workspaceRole.ID, "AWS").
	// 	First(&cspRoleMapping).Error; err != nil {
	// 	return nil, fmt.Errorf("failed to get CSP role mapping: %w", err)
	// }

	// STS 토큰 발급
	securityToken, err := getSecurityToken(cfg.WebIdentityToken, cspRoleMapping.RoleARN)
	if err != nil {
		return nil, fmt.Errorf("failed to get security token: %w", err)
	}

	// AWS 설정 로드
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 임시 자격 증명으로 새로운 AWS 설정 생성
	awsCfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		securityToken.AssumeRoleWithWebIdentityResult.Credentials.AccessKeyId,
		securityToken.AssumeRoleWithWebIdentityResult.Credentials.SecretAccessKey,
		securityToken.AssumeRoleWithWebIdentityResult.Credentials.SessionToken,
	))

	// IAM 클라이언트 생성
	client := iam.NewFromConfig(awsCfg)

	return &AWSIAMClient{
		client: client,
		config: cfg,
		db:     db,
	}, nil
}

// extractWorkspaceId 워크스페이스 티켓에서 워크스페이스 ID 추출
func extractWorkspaceId(ticket string) string {
	// JWT 토큰에서 페이로드 부분 추출
	parts := strings.Split(ticket, ".")
	if len(parts) != 3 {
		return ""
	}

	// Base64 디코딩
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	// JSON 파싱
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	return claims.Sub
}

// getSecurityToken STS 토큰 발급
func getSecurityToken(accessToken, roleArn string) (*AssumeRoleWithWebIdentityResponse, error) {
	// AWS STS 엔드포인트
	endpoint := os.Getenv("AWS_STS_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://sts.amazonaws.com"
	}

	// 요청 파라미터 설정
	params := url.Values{}
	params.Add("Action", "AssumeRoleWithWebIdentity")
	params.Add("Version", "2011-06-15")
	params.Add("WebIdentityToken", accessToken)

	params.Add("RoleArn", roleArn)
	//params.Add("RoleArn", "arn:aws:iam::050864702683:role/test-oidc-readonlyrole") // 고정된 role ARN 사용
	params.Add("RoleSessionName", "IAMManagerSession")
	params.Add("DurationSeconds", "3600")

	fmt.Printf("accessToken: %+v\n", accessToken)

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.URL.RawQuery = params.Encode()

	// 요청 전송
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("STS request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var result AssumeRoleWithWebIdentityResponse
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// CreateRole IAM 역할 생성
func (c *AWSIAMClient) CreateRole(ctx context.Context, role *csp.Role) error {
	// 정책 문서를 JSON 문자열로 변환
	policyDocument, err := json.Marshal(role.Policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	// 역할 생성 요청
	input := &iam.CreateRoleInput{
		RoleName:                 aws.String(role.Name),
		Description:              aws.String(role.Description),
		AssumeRolePolicyDocument: aws.String(string(policyDocument)),
		Tags:                     convertTags(role.Tags),
	}

	_, err = c.client.CreateRole(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// DeleteRole IAM 역할 삭제
func (c *AWSIAMClient) DeleteRole(ctx context.Context, roleName string) error {
	input := &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	}

	_, err := c.client.DeleteRole(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// GetRole IAM 역할 정보 조회
func (c *AWSIAMClient) GetRole(ctx context.Context, roleName string) (*csp.Role, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	result, err := c.client.GetRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// 정책 문서 파싱
	var policy csp.RolePolicy
	if err := json.Unmarshal([]byte(*result.Role.AssumeRolePolicyDocument), &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy document: %w", err)
	}

	return &csp.Role{
		Name:        *result.Role.RoleName,
		Description: *result.Role.Description,
		Policy:      &policy,
		Tags:        convertAWSTags(result.Role.Tags),
	}, nil
}

// UpdateRole IAM 역할 정보 수정
func (c *AWSIAMClient) UpdateRole(ctx context.Context, role *csp.Role) error {
	// 정책 문서를 JSON 문자열로 변환
	policyDocument, err := json.Marshal(role.Policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	// 역할 업데이트 요청
	input := &iam.UpdateRoleInput{
		RoleName:    aws.String(role.Name),
		Description: aws.String(role.Description),
	}

	_, err = c.client.UpdateRole(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	// 신뢰 정책 업데이트
	trustPolicyInput := &iam.UpdateAssumeRolePolicyInput{
		RoleName:       aws.String(role.Name),
		PolicyDocument: aws.String(string(policyDocument)),
	}

	_, err = c.client.UpdateAssumeRolePolicy(ctx, trustPolicyInput)
	if err != nil {
		return fmt.Errorf("failed to update assume role policy: %w", err)
	}

	return nil
}

// AttachRolePolicy IAM 역할에 정책 연결
func (c *AWSIAMClient) AttachRolePolicy(ctx context.Context, roleName string, policyArn string) error {
	input := &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	}

	_, err := c.client.AttachRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to attach role policy: %w", err)
	}

	return nil
}

// DetachRolePolicy IAM 역할에서 정책 분리
func (c *AWSIAMClient) DetachRolePolicy(ctx context.Context, roleName string, policyArn string) error {
	input := &iam.DetachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	}

	_, err := c.client.DetachRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to detach role policy: %w", err)
	}

	return nil
}

// ListRolePolicies IAM 역할에 연결된 정책 목록 조회
func (c *AWSIAMClient) ListRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	result, err := c.client.ListAttachedRolePolicies(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list role policies: %w", err)
	}

	policies := make([]string, 0, len(result.AttachedPolicies))
	for _, policy := range result.AttachedPolicies {
		policies = append(policies, *policy.PolicyName)
	}

	return policies, nil
}

// GetRolePolicy IAM 역할의 특정 정책 조회
func (c *AWSIAMClient) GetRolePolicy(ctx context.Context, roleName string, policyName string) (*csp.RolePolicy, error) {
	input := &iam.GetRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	result, err := c.client.GetRolePolicy(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get role policy: %w", err)
	}

	var policy csp.RolePolicy
	if err := json.Unmarshal([]byte(*result.PolicyDocument), &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy document: %w", err)
	}

	return &policy, nil
}

// PutRolePolicy IAM 역할에 정책 추가/수정
func (c *AWSIAMClient) PutRolePolicy(ctx context.Context, roleName string, policyName string, policy *csp.RolePolicy) error {
	// 정책 문서를 JSON 문자열로 변환
	policyDocument, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	input := &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(string(policyDocument)),
	}

	_, err = c.client.PutRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put role policy: %w", err)
	}

	return nil
}

// DeleteRolePolicy IAM 역할에서 정책 삭제
func (c *AWSIAMClient) DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error {
	input := &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	_, err := c.client.DeleteRolePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete role policy: %w", err)
	}

	return nil
}

// convertTags 일반 태그를 AWS 태그로 변환
func convertTags(tags map[string]string) []types.Tag {
	awsTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return awsTags
}

// convertAWSTags AWS 태그를 일반 태그로 변환
func convertAWSTags(awsTags []types.Tag) map[string]string {
	tags := make(map[string]string, len(awsTags))
	for _, tag := range awsTags {
		tags[*tag.Key] = *tag.Value
	}
	return tags
}
