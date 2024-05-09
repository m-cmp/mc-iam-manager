package stsmodule

type MciamCspCredentialsResponse struct {
	CspCredentials       []CspCredential `json:"cspCredentials"`
	UnSupportedProviders []string        `json:"unSupportedProviders"`
}

type CspCredential struct {
	Provider   string      `json:"provider"`
	Credential interface{} `json:"credential"`
}
