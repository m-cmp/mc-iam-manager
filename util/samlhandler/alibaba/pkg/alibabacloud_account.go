package saml2alibabacloud

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// AlibabaCloudAccount holds the AlibabaCloud account name and roles
type AlibabaCloudAccount struct {
	Name  string
	Roles []*RamRole
}

type RoleList struct {
	AccountAliasList map[string]string `json:"AccountAliasList"`
	RelayState       string            `json:"RelayState"`
	SAMLResponse     string            `json:"SamlResponse"`
	RoleInfoList     map[string][]struct {
		AccountId   string `json:"accountId"`
		ProviderArn string `json:"providerArn"`
		Raw         string `json:"raw"`
		RoleArn     string `json:"roleArn"`
		RoleName    string `json:"roleName"`
	}
}

// ParseAlibabaCloudAccounts extract the AlibabaCloud accounts from the saml assertion
func ParseAlibabaCloudAccounts(audience string, samlAssertion string) ([]*AlibabaCloudAccount, error) {
	res, err := http.PostForm(audience, url.Values{"SAMLResponse": {samlAssertion}})
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving AlibabaCloud login form")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving AlibabaCloud login body")
	}

	return ExtractAlibabaCloudAccounts(data)
}

// ExtractAlibabaCloudAccounts extract the accounts from the AlibabaCloud login html page
func ExtractAlibabaCloudAccounts(data []byte) ([]*AlibabaCloudAccount, error) {
	accounts := []*AlibabaCloudAccount{}

	html := string(data)
	if strings.Contains(html, "ROLE_SSO_PAGE:") {
		startIndex := strings.Index(html, "ROLE_SSO_PAGE:") + len("ROLE_SSO_PAGE:") + 1
		endIndex := startIndex + strings.Index(html[startIndex:], "\"}]}},") + 5
		roleListJson := html[startIndex:endIndex]

		var roleList RoleList
		if err := json.Unmarshal([]byte(roleListJson), &roleList); err != nil {
			return nil, errors.Wrap(err, "Role selection page response unmarshal error")
		}

		for accountId, accountRoleList := range roleList.RoleInfoList {
			account := new(AlibabaCloudAccount)
			account.Name = fmt.Sprintf("%s(%s)", roleList.AccountAliasList[accountId], accountId)
			for _, roleInfo := range accountRoleList {
				role := new(RamRole)
				role.Name = roleInfo.RoleName
				role.RoleARN = roleInfo.RoleArn
				role.PrincipalARN = roleInfo.ProviderArn
				account.Roles = append(account.Roles, role)
			}
			accounts = append(accounts, account)
		}
		return accounts, nil
	}

	return nil, errors.New("cannot find any roles")
}

// AssignPrincipals assign principal from roles
func AssignPrincipals(ramRoles []*RamRole, alibabacloudAccounts []*AlibabaCloudAccount) {

	principalARNs := make(map[string]string)
	for _, ramRole := range ramRoles {
		principalARNs[ramRole.RoleARN] = ramRole.PrincipalARN
	}

	for _, account := range alibabacloudAccounts {
		for _, ramRole := range account.Roles {
			ramRole.PrincipalARN = principalARNs[ramRole.RoleARN]
		}
	}

}

// LocateRole locate role by name
func LocateRole(ramRoles []*RamRole, roleName string) (*RamRole, error) {
	for _, ramRole := range ramRoles {
		if ramRole.RoleARN == roleName {
			return ramRole, nil
		}
	}

	return nil, fmt.Errorf("supplied `RoleArn` not found in saml assertion: %s", roleName)
}
