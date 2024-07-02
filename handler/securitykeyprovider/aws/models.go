package aws

var (
	AWSSecuritykeyInputParams = &AWSSecuritykeyInput{
		DurationSeconds: AWSSecuritykeyInputDurationSeconds,
		Action:          AWSSecuritykeyInputAction,
		Version:         AWSSecuritykeyInputVersion,
		RoleSessionName: AWSSecuritykeyInputRoleSessionName,
	}
)

// AWS sts 요청 파람
type AWSSecuritykeyInput struct {
	DurationSeconds  int    `json:"durationSeconds"`
	Action           string `json:"action"`
	Version          string `json:"version"`
	RoleSessionName  string `json:"roleSessionName"`
	RoleArn          string `json:"roleArn"`
	WebIdentityToken string `json:"webIdentityToken"`
}

// AWS sts 응답
type AssumeRoleWithWebIdentityResponse struct {
	AssumeRoleWithWebIdentityResult AssumeRoleWithWebIdentityResult `xml:"AssumeRoleWithWebIdentityResult"`
	ResponseMetadata                ResponseMetadata                `xml:"ResponseMetadata"`
}

type AssumeRoleWithWebIdentityResult struct {
	Audience               string          `xml:"Audience"`
	AssumedRoleUser        AssumedRoleUser `xml:"AssumedRoleUser"`
	Provider               string          `xml:"Provider"`
	Credentials            Credentials     `xml:"Credentials"`
	SubjectFromWebIdentity string          `xml:"SubjectFromWebIdentityToken"`
}

type AssumedRoleUser struct {
	AssumedRoleId string `xml:"AssumedRoleId"`
	Arn           string `xml:"Arn"`
}

type Credentials struct {
	AccessKeyId     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}

type ResponseMetadata struct {
	RequestId string `xml:"RequestId"`
}
