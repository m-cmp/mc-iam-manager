package service

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CspPolicyService CSP 정책 서비스
type CspPolicyService struct {
	db                  *gorm.DB
	cspPolicyRepo       *repository.CspPolicyRepository
	cspAccountRepo      *repository.CspAccountRepository
	cspRoleRepo         *repository.CspRoleRepository
	cspIdpConfigService *CspIdpConfigService
}

// NewCspPolicyService 새 CspPolicyService 인스턴스 생성
func NewCspPolicyService(db *gorm.DB, cspIdpConfigService *CspIdpConfigService) *CspPolicyService {
	return &CspPolicyService{
		db:                  db,
		cspPolicyRepo:       repository.NewCspPolicyRepository(db),
		cspAccountRepo:      repository.NewCspAccountRepository(db),
		cspRoleRepo:         repository.NewCspRoleRepository(db),
		cspIdpConfigService: cspIdpConfigService,
	}
}

// CreateCspPolicy CSP 정책 생성
func (s *CspPolicyService) CreateCspPolicy(req *model.CreateCspPolicyRequest) (*model.CspPolicy, error) {
	// CSP 계정 존재 확인
	account, err := s.cspAccountRepo.GetByID(req.CspAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("CSP account not found with ID: %d", req.CspAccountID)
	}

	// 이름 중복 확인
	exists, err := s.cspPolicyRepo.ExistsByNameAndAccountID(req.Name, req.CspAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check policy existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("policy with name '%s' already exists for this account", req.Name)
	}

	// 정책 생성
	policy := &model.CspPolicy{
		Name:         req.Name,
		CspAccountID: req.CspAccountID,
		PolicyType:   req.PolicyType,
		PolicyArn:    req.PolicyArn,
		PolicyDoc:    req.PolicyDoc,
		Description:  req.Description,
	}

	if err := s.cspPolicyRepo.Create(policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	log.Printf("Created CSP policy: %s (type: %s)", policy.Name, policy.PolicyType)
	return policy, nil
}

// GetCspPolicyByID ID로 CSP 정책 조회
func (s *CspPolicyService) GetCspPolicyByID(id uint) (*model.CspPolicy, error) {
	policy, err := s.cspPolicyRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy == nil {
		return nil, fmt.Errorf("policy not found with ID: %d", id)
	}
	return policy, nil
}

// ListCspPolicies CSP 정책 목록 조회
func (s *CspPolicyService) ListCspPolicies(filter *model.CspPolicyFilter) ([]*model.CspPolicy, error) {
	policies, err := s.cspPolicyRepo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	return policies, nil
}

// UpdateCspPolicy CSP 정책 수정
func (s *CspPolicyService) UpdateCspPolicy(id uint, req *model.UpdateCspPolicyRequest) (*model.CspPolicy, error) {
	// 기존 정책 조회
	policy, err := s.cspPolicyRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy == nil {
		return nil, fmt.Errorf("policy not found with ID: %d", id)
	}

	// 필드 업데이트
	if req.Name != "" {
		// 이름 변경 시 중복 확인
		if req.Name != policy.Name {
			exists, err := s.cspPolicyRepo.ExistsByNameAndAccountID(req.Name, policy.CspAccountID)
			if err != nil {
				return nil, fmt.Errorf("failed to check policy existence: %w", err)
			}
			if exists {
				return nil, fmt.Errorf("policy with name '%s' already exists", req.Name)
			}
		}
		policy.Name = req.Name
	}
	if req.PolicyArn != "" {
		policy.PolicyArn = req.PolicyArn
	}
	if req.PolicyDoc != nil {
		policy.PolicyDoc = req.PolicyDoc
	}
	if req.Description != "" {
		policy.Description = req.Description
	}

	if err := s.cspPolicyRepo.Update(policy); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	log.Printf("Updated CSP policy: %s (ID: %d)", policy.Name, policy.ID)
	return policy, nil
}

// DeleteCspPolicy CSP 정책 삭제
func (s *CspPolicyService) DeleteCspPolicy(id uint) error {
	// 정책 존재 확인
	exists, err := s.cspPolicyRepo.ExistsByID(id)
	if err != nil {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("policy not found with ID: %d", id)
	}

	// 연결된 역할 확인
	roles, err := s.cspPolicyRepo.GetRolesByPolicyID(id)
	if err != nil {
		return fmt.Errorf("failed to get roles by policy: %w", err)
	}
	if len(roles) > 0 {
		return fmt.Errorf("cannot delete policy: %d roles are attached", len(roles))
	}

	if err := s.cspPolicyRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	log.Printf("Deleted CSP policy with ID: %d", id)
	return nil
}

// AttachPolicyToRole 역할에 정책 연결
func (s *CspPolicyService) AttachPolicyToRole(roleID, policyID uint) error {
	// 역할 존재 확인
	roleExists, err := s.cspRoleRepo.ExistsCspRoleByID(roleID)
	if err != nil {
		return fmt.Errorf("failed to check role existence: %w", err)
	}
	if !roleExists {
		return fmt.Errorf("CSP role not found with ID: %d", roleID)
	}

	// 정책 존재 확인
	policyExists, err := s.cspPolicyRepo.ExistsByID(policyID)
	if err != nil {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}
	if !policyExists {
		return fmt.Errorf("CSP policy not found with ID: %d", policyID)
	}

	// 이미 연결되어 있는지 확인
	attached, err := s.cspPolicyRepo.IsPolicyAttachedToRole(roleID, policyID)
	if err != nil {
		return fmt.Errorf("failed to check policy attachment: %w", err)
	}
	if attached {
		return fmt.Errorf("policy is already attached to the role")
	}

	if err := s.cspPolicyRepo.AttachPolicyToRole(roleID, policyID); err != nil {
		return fmt.Errorf("failed to attach policy to role: %w", err)
	}

	log.Printf("Attached policy %d to role %d", policyID, roleID)
	return nil
}

// DetachPolicyFromRole 역할에서 정책 분리
func (s *CspPolicyService) DetachPolicyFromRole(roleID, policyID uint) error {
	// 연결 여부 확인
	attached, err := s.cspPolicyRepo.IsPolicyAttachedToRole(roleID, policyID)
	if err != nil {
		return fmt.Errorf("failed to check policy attachment: %w", err)
	}
	if !attached {
		return fmt.Errorf("policy is not attached to the role")
	}

	if err := s.cspPolicyRepo.DetachPolicyFromRole(roleID, policyID); err != nil {
		return fmt.Errorf("failed to detach policy from role: %w", err)
	}

	log.Printf("Detached policy %d from role %d", policyID, roleID)
	return nil
}

// GetPoliciesByRoleID 역할에 연결된 정책 목록 조회
func (s *CspPolicyService) GetPoliciesByRoleID(roleID uint) ([]*model.CspPolicy, error) {
	policies, err := s.cspPolicyRepo.GetPoliciesByRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by role: %w", err)
	}
	return policies, nil
}

// SyncPoliciesFromCloud CSP에서 정책 동기화
func (s *CspPolicyService) SyncPoliciesFromCloud(ctx context.Context, req *model.SyncPoliciesRequest) ([]*model.CspPolicy, error) {
	// CSP 계정 조회
	account, err := s.cspAccountRepo.GetByID(req.CspAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("CSP account not found with ID: %d", req.CspAccountID)
	}

	switch account.CspType {
	case "aws":
		return s.syncAwsPolicies(ctx, account, req.PolicyScope)
	case "gcp":
		return nil, fmt.Errorf("GCP policy sync not implemented yet")
	case "azure":
		return nil, fmt.Errorf("Azure policy sync not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported CSP type: %s", account.CspType)
	}
}

// syncAwsPolicies AWS에서 정책 동기화
func (s *CspPolicyService) syncAwsPolicies(ctx context.Context, account *model.CspAccount, scope string) ([]*model.CspPolicy, error) {
	// IDP 설정을 통해 임시 자격 증명 획득
	idpConfigs, err := s.cspIdpConfigService.GetActiveIdpConfigsByAccountID(account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDP configs: %w", err)
	}
	if len(idpConfigs) == 0 {
		return nil, fmt.Errorf("no active IDP config found for account")
	}

	// 첫 번째 활성 IDP 설정 사용
	idpConfig := idpConfigs[0]

	// 임시 자격 증명 획득
	tempCred, err := s.cspIdpConfigService.AssumeRoleWithIdpConfig(ctx, idpConfig.ID,
		idpConfig.Config["role_arn"],
		"mciam-policy-sync",
		3600)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	// AWS IAM 클라이언트 생성
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(tempCred.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			tempCred.AccessKeyId,
			tempCred.SecretAccessKey,
			tempCred.SessionToken,
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	iamClient := iam.NewFromConfig(cfg)

	// 정책 목록 조회
	var policyScope iamtypes.PolicyScopeType
	switch scope {
	case "All":
		policyScope = iamtypes.PolicyScopeTypeAll
	case "AWS":
		policyScope = iamtypes.PolicyScopeTypeAws
	case "Local":
		policyScope = iamtypes.PolicyScopeTypeLocal
	default:
		policyScope = iamtypes.PolicyScopeTypeLocal
	}

	input := &iam.ListPoliciesInput{
		Scope: policyScope,
	}

	var syncedPolicies []*model.CspPolicy
	paginator := iam.NewListPoliciesPaginator(iamClient, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list AWS policies: %w", err)
		}

		for _, awsPolicy := range page.Policies {
			// 기존 정책 확인
			existingPolicy, _ := s.cspPolicyRepo.GetByArn(*awsPolicy.Arn)
			if existingPolicy != nil {
				// 업데이트
				existingPolicy.Name = *awsPolicy.PolicyName
				if awsPolicy.Description != nil {
					existingPolicy.Description = *awsPolicy.Description
				}
				if err := s.cspPolicyRepo.Update(existingPolicy); err != nil {
					log.Printf("Failed to update policy %s: %v", *awsPolicy.PolicyName, err)
					continue
				}
				syncedPolicies = append(syncedPolicies, existingPolicy)
			} else {
				// 새로 생성
				newPolicy := &model.CspPolicy{
					Name:         *awsPolicy.PolicyName,
					CspAccountID: account.ID,
					PolicyType:   model.PolicyTypeManaged,
					PolicyArn:    *awsPolicy.Arn,
				}
				if awsPolicy.Description != nil {
					newPolicy.Description = *awsPolicy.Description
				}
				if err := s.cspPolicyRepo.Create(newPolicy); err != nil {
					log.Printf("Failed to create policy %s: %v", *awsPolicy.PolicyName, err)
					continue
				}
				syncedPolicies = append(syncedPolicies, newPolicy)
			}
		}
	}

	log.Printf("Synced %d policies from AWS", len(syncedPolicies))
	return syncedPolicies, nil
}

// GetPolicyDocument CSP에서 정책 문서 조회
func (s *CspPolicyService) GetPolicyDocument(ctx context.Context, policyID uint) (map[string]interface{}, error) {
	policy, err := s.cspPolicyRepo.GetByID(policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy == nil {
		return nil, fmt.Errorf("policy not found with ID: %d", policyID)
	}

	// 이미 PolicyDoc가 있으면 반환
	if policy.PolicyDoc != nil {
		return policy.PolicyDoc, nil
	}

	// 관리형 정책이고 ARN이 있으면 CSP에서 조회
	if policy.PolicyType == model.PolicyTypeManaged && policy.PolicyArn != "" {
		account, err := s.cspAccountRepo.GetByID(policy.CspAccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get CSP account: %w", err)
		}

		if account.CspType == "aws" {
			return s.getAwsPolicyDocument(ctx, account, policy.PolicyArn)
		}
	}

	return nil, fmt.Errorf("policy document not available")
}

// getAwsPolicyDocument AWS에서 정책 문서 조회
func (s *CspPolicyService) getAwsPolicyDocument(ctx context.Context, account *model.CspAccount, policyArn string) (map[string]interface{}, error) {
	// IDP 설정을 통해 임시 자격 증명 획득
	idpConfigs, err := s.cspIdpConfigService.GetActiveIdpConfigsByAccountID(account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDP configs: %w", err)
	}
	if len(idpConfigs) == 0 {
		return nil, fmt.Errorf("no active IDP config found for account")
	}

	idpConfig := idpConfigs[0]
	tempCred, err := s.cspIdpConfigService.AssumeRoleWithIdpConfig(ctx, idpConfig.ID,
		idpConfig.Config["role_arn"],
		"mciam-policy-get",
		900)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	// AWS IAM 클라이언트 생성
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(tempCred.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			tempCred.AccessKeyId,
			tempCred.SecretAccessKey,
			tempCred.SessionToken,
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	iamClient := iam.NewFromConfig(cfg)

	// 정책 정보 조회
	getPolicyInput := &iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	}
	policyResult, err := iamClient.GetPolicy(ctx, getPolicyInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// 정책 버전 문서 조회
	getPolicyVersionInput := &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(policyArn),
		VersionId: policyResult.Policy.DefaultVersionId,
	}
	versionResult, err := iamClient.GetPolicyVersion(ctx, getPolicyVersionInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy version: %w", err)
	}

	// URL 디코딩 및 JSON 파싱
	if versionResult.PolicyVersion.Document != nil {
		// AWS는 URL 인코딩된 JSON을 반환
		// 여기서는 간단히 string으로 반환 (실제로는 파싱 필요)
		return map[string]interface{}{
			"document": *versionResult.PolicyVersion.Document,
		}, nil
	}

	return nil, fmt.Errorf("policy document not found")
}

// GetManagedPoliciesByAccountID 특정 계정의 관리형 정책 목록 조회
func (s *CspPolicyService) GetManagedPoliciesByAccountID(accountID uint) ([]*model.CspPolicy, error) {
	policies, err := s.cspPolicyRepo.GetManagedPoliciesByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed policies: %w", err)
	}
	return policies, nil
}

// GetPoliciesByAccountID 특정 계정의 모든 정책 목록 조회
func (s *CspPolicyService) GetPoliciesByAccountID(accountID uint) ([]*model.CspPolicy, error) {
	policies, err := s.cspPolicyRepo.GetByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by account: %w", err)
	}
	return policies, nil
}
