package securitykeyprovider

type Alibaba struct{}

type AWS struct{}

type CspCredential struct {
	Provider   string      `json:"provider"`
	Credential interface{} `json:"credential"`
}

type MciamCspCredentialsResponse struct {
	CspCredentials       []CspCredential `json:"cspCredentials"`
	UnSupportedProviders []string        `json:"unSupportedProviders"`
}
