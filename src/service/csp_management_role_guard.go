package service

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
)

// 관리용 Role 보호 (계획 3단계)
// IdP Config 의 role_arn 은 IAM 이 AWS 에 쓸 때 assume 하는 관리용 Role 이다. 이 Role 을 사용자 매핑 대상으로 쓰거나
// IAM 을 통해 바꾸거나 지우면 권한 상승이나 IAM 기능 전체 정지로 이어진다. 활성 여부와 관계없이 모든 Config 가 대상이다.
// 매핑 해제(DELETE /roles/csp-roles)는 막지 않는다 (기존 설치에 남은 관리용 Role 매핑을 정리하는 경로).

// isManagementRole 이름 또는 ARN 이 어떤 IdP Config 의 관리용 Role(role_arn)인지 확인한다.
// ARN 이 있으면 ARN 전체로, 없으면 Role 이름(ARN 의 마지막 경로)으로 비교한다. 이름 비교는 다른 계정의 같은 이름도 보호 쪽으로 판정한다.
func (s *CspRoleService) isManagementRole(roleName, roleArn string) (bool, *model.CspIdpConfig, error) {
	configs, err := s.cspIdpConfigRepo.List(nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to list IdP configs: %w", err)
	}
	for _, cfg := range configs {
		if cfg.Config == nil {
			continue
		}
		mgmtArn := cfg.Config["role_arn"]
		if mgmtArn == "" {
			continue
		}
		if (roleArn != "" && roleArn == mgmtArn) || (roleName != "" && roleNameFromArn(mgmtArn) == roleName) {
			return true, cfg, nil
		}
	}
	return false, nil, nil
}

// GuardNotManagementRole 관리용 Role 이면 CspRoleProtectedError(핸들러는 409)를 돌려준다.
// action 은 오류 문구에 쓴다 (예: "mapped", "updated", "deleted").
// CspRole 매핑·등록·수정·삭제와 이후 정책·신뢰 정책 변경 경로에서 같은 함수로 확인한다.
func (s *CspRoleService) GuardNotManagementRole(roleName, roleArn, action string) error {
	protected, cfg, err := s.isManagementRole(roleName, roleArn)
	if err != nil {
		return err
	}
	if !protected {
		return nil
	}
	label := roleName
	if label == "" {
		label = roleArn
	}
	return &CspRoleProtectedError{Msg: fmt.Sprintf("CSP role %s is the management role of IdP config %d (%s) and cannot be %s", label, cfg.ID, cfg.Name, action)}
}

// GuardCspRoleNotManagementRole 저장된 CspRole 레코드 기준으로 확인한다.
func (s *CspRoleService) GuardCspRoleNotManagementRole(role *model.CspRole, action string) error {
	if role == nil {
		return nil
	}
	return s.GuardNotManagementRole(role.Name, role.IamIdentifier, action)
}
