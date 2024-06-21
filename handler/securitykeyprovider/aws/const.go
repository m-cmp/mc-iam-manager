package aws

const (
	AWSSecuritykeyEndPoint = "https://sts.amazonaws.com/"

	AWSSecuritykeyInputDurationSeconds = 900 // 900초 기본설정, accesstoken 과 별도 라이프사이클을 가짐.
	AWSSecuritykeyInputAction          = "AssumeRoleWithWebIdentity"
	AWSSecuritykeyInputVersion         = "2011-06-15"
	AWSSecuritykeyInputRoleSessionName = "web-identity-federation"
)
