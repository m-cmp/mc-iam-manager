package saml2aws

import (
	"sort"

	"mc_iam_manager/util/samlhandler/aws/pkg/cfg"
	"mc_iam_manager/util/samlhandler/aws/pkg/creds"
	"mc_iam_manager/util/samlhandler/provider/keycloak"
)

// ProviderList list of providers with their MFAs
type ProviderList map[string][]string

// MFAsByProvider a list of providers with their respective supported MFAs
var MFAsByProvider = ProviderList{
	"AzureAD":       []string{"Auto", "PhoneAppOTP", "PhoneAppNotification", "OneWaySMS"},
	"ADFS":          []string{"Auto", "VIP", "Azure", "Defender"},
	"ADFS2":         []string{"Auto", "RSA"}, // nothing automatic about ADFS 2.x
	"Ping":          []string{"Auto"},        // automatically detects PingID
	"PingOne":       []string{"Auto"},        // automatically detects PingID
	"JumpCloud":     []string{"Auto", "TOTP", "WEBAUTHN", "DUO", "PUSH"},
	"Okta":          []string{"Auto", "PUSH", "DUO", "SMS", "TOTP", "OKTA", "FIDO", "YUBICO TOKEN:HARDWARE", "SYMANTEC"}, // automatically detects DUO, SMS, ToTP, and FIDO
	"OneLogin":      []string{"Auto", "OLP", "SMS", "TOTP", "YUBIKEY"},                                                   // automatically detects OneLogin Protect, SMS and ToTP
	"Authentik":     []string{"Auto"},
	"KeyCloak":      []string{"Auto"}, // automatically detects ToTP
	"GoogleApps":    []string{"Auto"}, // automatically detects ToTP
	"Shibboleth":    []string{"Auto", "None"},
	"F5APM":         []string{"Auto"},
	"Akamai":        []string{"Auto", "DUO", "SMS", "EMAIL", "TOTP"},
	"ShibbolethECP": []string{"auto", "phone", "push", "passcode"},
	"NetIQ":         []string{"Auto", "Privileged"},
	"Browser":       []string{"Auto"},
	"Auth0":         []string{"Auto"},
}

// Names get a list of provider names
func (mfbp ProviderList) Names() []string {
	keys := []string{}
	for k := range mfbp {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// Mfas retrieve a sorted list of mfas from the provider list
func (mfbp ProviderList) Mfas(provider string) []string {
	mfas := mfbp[provider]

	sort.Strings(mfas)

	return mfas
}

func (mfbp ProviderList) stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func invalidMFA(provider string, mfa string) bool {
	supportedMfas := MFAsByProvider.Mfas(provider)
	return !MFAsByProvider.stringInSlice(mfa, supportedMfas)
}

// SAMLClient client interface
type SAMLClient interface {
	Authenticate(loginDetails *creds.LoginDetails) (string, error)
	Validate(loginDetails *creds.LoginDetails) error
}

// NewSAMLClient create a new SAML client
func NewSAMLClient(idpAccount *cfg.IDPAccount) (SAMLClient, error) {
	return keycloak.New(idpAccount)
}
