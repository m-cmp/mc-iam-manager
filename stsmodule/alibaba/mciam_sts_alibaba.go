package mciam_sts_alibaba

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AlibabaSecuritykeyEndPoint = "https://sts.cn-beijing.aliyuncs.com"
)

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

func GetAlibabaSecurityToken(c buffalo.Context) (AlibabaAssumeRoleWithOIDCResponse, error) {

	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
	userId := accesskeyDecode(accessToken)["preferred_username"].(string)

	var securityToken AlibabaAssumeRoleWithOIDCResponse

	inputParams := AlibabaSecuritykeyParamInput{}
	inputParams.OIDCProviderArn = os.Getenv("AlibabaOIDCProviderArn") // TODO: 우선 하드코딩.. keyclaok 또는 자체에서 DB 로 처리 예정..
	inputParams.RoleArn = os.Getenv("AlibabaRoleArn")                 // TODO: 우선 하드코딩.. keyclaok 또는 자체에서 DB 로 처리 예정..
	inputParams.OIDCToken = accessToken
	inputParams.RoleSessionName = userId
	inputParams.Timestamp = timeStampNowISO8601()

	urlValues, err := structToUrlValues(inputParams)
	if err != nil {
		return securityToken, err
	}

	req, err := http.NewRequest("POST", AlibabaSecuritykeyEndPoint, bytes.NewBufferString(urlValues.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return securityToken, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-acs-version", "2015-04-01")
	req.Header.Set("x-acs-action", "AssumeRoleWithOIDC")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return securityToken, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return securityToken, errors.New(resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return securityToken, err
	}

	Unmarshalerr := xml.Unmarshal(responseBody, &securityToken)
	if Unmarshalerr != nil {
		fmt.Println("XML unmarshaling error:", Unmarshalerr)
		return securityToken, err
	}

	return securityToken, nil
}

func structToUrlValues(s interface{}) (url.Values, error) {
	values := url.Values{}
	// 구조체 순회
	v := reflect.ValueOf(s)
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		fieldValue := v.Field(i).Interface()
		values.Add(fieldName, fmt.Sprintf("%v", fieldValue))
	}
	return values, nil
}

func timeStampNowISO8601() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}

func accesskeyDecode(jwtToken string) jwt.MapClaims {
	claims := jwt.MapClaims{}
	jwt.ParseWithClaims(jwtToken, claims, func(token *jwt.Token) (interface{}, error) { return "", nil })
	return claims
}
