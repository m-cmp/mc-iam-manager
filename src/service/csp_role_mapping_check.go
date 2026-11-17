package service

import (
	"fmt"
	"strings"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
)

// ValidateCspRoleForMapping 역할 Master(csp Sub) ↔ CspRole 매핑 전에 대상 CspRole 이 발급에 쓸 수 있는 상태인지 검사한다.
// 소속 IdP Config 가 있고, 매핑 인증 방식·CSP 타입이 그 Config 와 같고, idp/iam identifier 가 채워져 있어야 한다.
func (s *CspRoleService) ValidateCspRoleForMapping(cspRoleID uint, authMethod constants.AuthMethod, cspType string) error {
	role, err := s.cspRoleRepo.GetRoleByID(cspRoleID)
	if err != nil {
		return err
	}
	var cfg *model.CspIdpConfig
	if role.CspIdpConfigID != nil {
		cfg, err = s.cspIdpConfigRepo.GetByID(*role.CspIdpConfigID)
		if err != nil {
			return err
		}
	}
	return validateCspRoleForMapping(role, cfg, authMethod, cspType)
}

func validateCspRoleForMapping(role *model.CspRole, cfg *model.CspIdpConfig, authMethod constants.AuthMethod, cspType string) error {
	if authMethod == "" {
		authMethod = constants.AuthMethodOIDC
	}
	var problems []string
	if cfg == nil {
		problems = append(problems, "IdP config is not linked (csp_idp_config_id)")
	} else if string(cfg.AuthMethod) != string(authMethod) {
		problems = append(problems, fmt.Sprintf("authMethod %s does not match IdP config authMethod %s", authMethod, cfg.AuthMethod))
	}
	if cspType != "" && role.CspType != cspType {
		problems = append(problems, fmt.Sprintf("cspType %s does not match CSP role cspType %s", cspType, role.CspType))
	}
	if role.IdpIdentifier == "" {
		problems = append(problems, "idp_identifier is empty")
	}
	if role.IamIdentifier == "" {
		problems = append(problems, "iam_identifier is empty")
	}
	if len(problems) > 0 {
		return cspRoleRequestErrorf("CSP role %d (%s) cannot be mapped: %s", role.ID, role.Name, strings.Join(problems, "; "))
	}
	return nil
}
