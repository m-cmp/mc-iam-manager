package alibaba

import "encoding/xml"

// Alibaba sts 요청 파람
type AlibabaSecuritykeyParamInput struct {
	OIDCProviderArn string
	RoleArn         string
	OIDCToken       string
	RoleSessionName string
	Timestamp       string
}

type AlibabaAssumeRoleWithOIDCResponse struct {
	XMLName       xml.Name `xml:"AssumeRoleWithOIDCResponse"`
	RequestId     string   `xml:"RequestId"`
	OIDCTokenInfo struct {
		Issuer           string `xml:"Issuer"`
		IssuanceTime     string `xml:"IssuanceTime"`
		VerificationInfo string `xml:"VerificationInfo"`
		ExpirationTime   string `xml:"ExpirationTime"`
		Subject          string `xml:"Subject"`
		ClientIds        string `xml:"ClientIds"`
	} `xml:"OIDCTokenInfo"`
	AssumedRoleUser struct {
		Arn           string `xml:"Arn"`
		AssumedRoleId string `xml:"AssumedRoleId"`
	} `xml:"AssumedRoleUser"`
	Credentials struct {
		SecurityToken   string `xml:"SecurityToken"`
		AccessKeyId     string `xml:"AccessKeyId"`
		AccessKeySecret string `xml:"AccessKeySecret"`
		Expiration      string `xml:"Expiration"`
	} `xml:"Credentials"`
}
