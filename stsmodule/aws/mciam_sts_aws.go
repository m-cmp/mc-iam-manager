package mciam_sts_aws

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/gobuffalo/buffalo"
)

const (
	AWSSecuritykeyEndPoint = "https://sts.amazonaws.com/"
)

var (
	AWSSecuritykeyInputParams = &AWSSecuritykeyInput{
		DurationSeconds: 900, // 900초 기본설정, accesstoken 과 별도 라이프사이클을 가짐.
		Action:          "AssumeRoleWithWebIdentity",
		Version:         "2011-06-15",
		RoleSessionName: "web-identity-federation",
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

func GetAwsSecurityToken(c buffalo.Context) (AssumeRoleWithWebIdentityResponse, error) {

	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	var securityToken AssumeRoleWithWebIdentityResponse

	inputParams := AWSSecuritykeyInputParams
	inputParams.RoleArn = os.Getenv("AWSRoleArn") // TODO: 우선 하드코딩.. keyclaok 또는 자체에서 DB 로 처리 예정..
	inputParams.WebIdentityToken = accessToken

	encodedinputParams, err := structToMap(*inputParams)
	if err != nil {
		return securityToken, err
	}

	req, err := http.NewRequest("GET", AWSSecuritykeyEndPoint, nil)
	if err != nil {
		return securityToken, err
	}

	q := req.URL.Query()
	for key, value := range encodedinputParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return securityToken, errors.New(resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return securityToken, err
	}

	err = xml.Unmarshal([]byte(string(respBody)), &securityToken)
	if err != nil {
		return securityToken, err
	}
	return securityToken, nil
}

func structToMap(s interface{}) (map[string]string, error) {
	result := make(map[string]string)

	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface()) {
			continue
		}

		result[field.Name] = fmt.Sprintf("%v", value.Interface())
	}

	return result, nil
}
