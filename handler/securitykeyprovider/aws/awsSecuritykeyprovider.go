package aws

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/m-cmp/mc-iam-manager/handler/securitykeyprovider"

	"github.com/gobuffalo/buffalo"
)

type AWS struct{}

func (AWS) GetSecurityKey(c buffalo.Context) (*securitykeyprovider.CspCredential, error) {
	var result securitykeyprovider.CspCredential
	result.Provider = "AWS"

	var securityToken AssumeRoleWithWebIdentityResponse
	accessToken := c.Value("accessToken").(string)

	inputParams := AWSSecuritykeyInputParams
	inputParams.WebIdentityToken = accessToken

	inputParams.RoleArn = os.Getenv("AWSRoleArn") // TODO: 우선 하드코딩.. keyclaok 또는 자체에서 DB 로 처리 예정..
	encodedinputParams, err := StructToMap(*inputParams)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", AWSSecuritykeyEndPoint, nil)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("response is %s. check aws iam settings...", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal([]byte(string(respBody)), &securityToken)
	if err != nil {
		return nil, err
	}

	result.Credential = securityToken.AssumeRoleWithWebIdentityResult.Credentials
	return &result, nil
}
