package alibaba

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"

	"mc-iam-manager/handler/securitykeyprovider"

	"github.com/gobuffalo/buffalo"
)

type Alibaba struct{}

func (Alibaba) GetSecurityKey(c buffalo.Context) (*securitykeyprovider.CspCredential, error) {
	var result securitykeyprovider.CspCredential
	result.Provider = "Alibaba"

	accessToken := c.Value("accessToken").(string)
	userId := c.Value("preferredUsername").(string)

	inputParams := AlibabaSecuritykeyParamInput{}
	inputParams.OIDCProviderArn = os.Getenv("AlibabaOIDCProviderArn") // TODO: 우선 하드코딩.. keyclaok 또는 자체에서 DB 로 처리 예정..
	inputParams.RoleArn = os.Getenv("AlibabaRoleArn")                 // TODO: 우선 하드코딩.. keyclaok 또는 자체에서 DB 로 처리 예정..
	inputParams.OIDCToken = accessToken
	inputParams.RoleSessionName = userId
	inputParams.Timestamp = TimeStampNowISO8601()

	urlValues, err := StructToUrlValues(inputParams)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", AlibabaSecuritykeyEndPoint, bytes.NewBufferString(urlValues.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-acs-version", HeaderxacsVersion)
	req.Header.Set("x-acs-action", HeaderxacsAction)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response is %s. check alibaba iam settings...", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var securityToken AlibabaAssumeRoleWithOIDCResponse
	Unmarshalerr := xml.Unmarshal(responseBody, &securityToken)
	if Unmarshalerr != nil {
		fmt.Println("XML unmarshaling error:", Unmarshalerr)
		return nil, err
	}
	result.Credential = securityToken.Credentials
	return &result, nil
}
