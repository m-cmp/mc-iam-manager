package saml2alibabacloud

import (
	"fmt"
	"strings"
)

// RamRole AlibabaCloud RAM role attributes
type RamRole struct {
	RoleARN      string
	PrincipalARN string
	Name         string
}

// ParseRamRoles parses and splits the roles while also validating the contents
func ParseRamRoles(roles []string) ([]*RamRole, error) {
	ramRoles := make([]*RamRole, len(roles))

	for i, role := range roles {
		ramRole, err := parseRole(role)
		if err != nil {
			return nil, err
		}

		ramRoles[i] = ramRole
	}

	return ramRoles, nil
}

func parseRole(role string) (*RamRole, error) {
	tokens := strings.Split(role, ",")

	if len(tokens) != 2 {
		return nil, fmt.Errorf("invalid role string only %d tokens", len(tokens))
	}

	ramRole := &RamRole{}

	for _, token := range tokens {
		if strings.Contains(token, ":saml-provider") {
			ramRole.PrincipalARN = strings.TrimSpace(token)
		}
		if strings.Contains(token, ":role") {
			ramRole.RoleARN = strings.TrimSpace(token)
		}
	}

	if ramRole.PrincipalARN == "" {
		return nil, fmt.Errorf("unable to locate `PrincipalARN` in: %s", role)
	}

	if ramRole.RoleARN == "" {
		return nil, fmt.Errorf("unable to locate `RoleARN` in: %s", role)
	}

	return ramRole, nil
}
