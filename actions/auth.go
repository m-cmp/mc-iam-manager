package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/stsmodule"
	alibabaStsModule "mc_iam_manager/stsmodule/alibaba"
	awsStsModule "mc_iam_manager/stsmodule/aws"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
)

var (
	keycloakHost         string
	keycloakRealm        string
	keycloakClient       string
	keycloakClientSecret string
)

func init() {
	keycloakHost = os.Getenv("keycloakHost")
	keycloakRealm = os.Getenv("keycloakRealm")
	keycloakClient = os.Getenv("keycloakClient")
	keycloakClientSecret = os.Getenv("keycloakClientSecret")
}

func AuthLoginHandler(c buffalo.Context) error {

	user := &iammodels.UserLogin{}
	if err := c.Bind(user); err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{
				"code": err.Error(),
				"msg":  "user input bind Err",
			}))
	}

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.Id, Name: "id"},
		&validators.StringIsPresent{Field: user.Password, Name: "password"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	formData := url.Values{
		"username":      {user.Id},
		"password":      {user.Password},
		"client_id":     {keycloakClient},
		"client_secret": {keycloakClientSecret},
		"grant_type":    {"password"},
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   keycloakHost,
	}
	tokenPath := "/realms/" + keycloakRealm + "/protocol/openid-connect/token"
	tokenEndpoint := baseURL.ResolveReference(&url.URL{Path: tokenPath})

	req, _ := http.NewRequest("POST", tokenEndpoint.String(), strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Render(resp.StatusCode,
			r.JSON(map[string]string{"code": resp.Status}))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	var accessTokenResponse iammodels.KeycloakAccessTokenResponse
	jsonerr := json.Unmarshal(respBody, &accessTokenResponse)
	if jsonerr != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": jsonerr.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(accessTokenResponse))
}

func AuthLoginRefreshHandler(c buffalo.Context) error {

	user := &iammodels.UserLoginRefresh{}
	if err := c.Bind(user); err != nil {
		return c.Render(http.StatusBadRequest,
			r.JSON(map[string]string{
				"code": err.Error(),
				"msg":  "user input bind Err",
			}))
	}

	user.AccessToken = c.Request().Header.Get("Authorization")

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.AccessToken, Name: "Authorization"},
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	formData := url.Values{
		"client_id":     {keycloakClient},
		"client_secret": {keycloakClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {user.RefreshToken},
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   keycloakHost,
	}
	tokenPath := "/realms/" + keycloakRealm + "/protocol/openid-connect/token"
	tokenEndpoint := baseURL.ResolveReference(&url.URL{Path: tokenPath})

	req, _ := http.NewRequest("POST", tokenEndpoint.String(), strings.NewReader(formData.Encode()))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", user.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Render(resp.StatusCode,
			r.JSON(map[string]string{"code": resp.Status}))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	var accessTokenResponse iammodels.KeycloakAccessTokenResponse
	jsonerr := json.Unmarshal(respBody, &accessTokenResponse)
	if jsonerr != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": jsonerr.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(accessTokenResponse))
}

func AuthLogoutHandler(c buffalo.Context) error {

	user := &iammodels.UserLogout{}
	user.AccessToken = c.Request().Header.Get("Authorization")
	if err := c.Bind(user); err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{
				"code": err.Error(),
				"msg":  "user input bind Err",
			}))
	}

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.AccessToken, Name: "Authorization"},
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	formData := url.Values{
		"client_id":     {keycloakClient},
		"client_secret": {keycloakClientSecret},
		"refresh_token": {user.RefreshToken},
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   keycloakHost,
	}

	endSessionPath := "/realms/" + keycloakRealm + "/protocol/openid-connect/logout"
	endSessionEndpoint := baseURL.ResolveReference(&url.URL{Path: endSessionPath})

	req, err := http.NewRequest("POST", endSessionEndpoint.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", user.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	defer resp.Body.Close()

	if resp.Status != "204 No Content" {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"code": resp.Status}))
	}

	return c.Render(http.StatusOK, nil)
}

func AuthGetUserValidate(c buffalo.Context) error {
	accessToken := c.Request().Header.Get("Authorization")

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   keycloakHost,
	}
	getUserInfoPath := "/realms/" + keycloakRealm + "/protocol/openid-connect/userinfo"
	getUserInfoEndpoint := baseURL.ResolveReference(&url.URL{Path: getUserInfoPath})

	req, err := http.NewRequest("GET", getUserInfoEndpoint.String(), nil)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	req.Header.Set("Authorization", accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Render(resp.StatusCode,
			r.JSON(map[string]string{"code": resp.Status}))
	}

	return c.Render(http.StatusOK, nil)
}

func AuthGetUserInfo(c buffalo.Context) error {
	accessToken := c.Request().Header.Get("Authorization")

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   keycloakHost,
	}
	getUserInfoPath := "/realms/" + keycloakRealm + "/protocol/openid-connect/userinfo"
	getUserInfoEndpoint := baseURL.ResolveReference(&url.URL{Path: getUserInfoPath})

	req, err := http.NewRequest("GET", getUserInfoEndpoint.String(), nil)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	req.Header.Set("Authorization", accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Render(resp.StatusCode,
			r.JSON(map[string]string{"code": resp.Status}))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	var userinfo map[string]interface{}
	if err := json.Unmarshal([]byte(respBody), &userinfo); err != nil {
		fmt.Println("JSON 파싱 에러:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(userinfo))
}

func AuthGetSecurityKeyHandler(c buffalo.Context) error {
	accessToken := strings.Replace(c.Request().Header.Get("Authorization"), "Bearer ", "", -1)
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	providers := c.Param("providers")
	var providerarr []string
	if providers != "" {
		providerarr = strings.Split(providers, ",")
	} else {
		providerarr = []string{"aws", "alibaba"}
	}

	var mciamCspCredentialsResponse stsmodule.MciamCspCredentialsResponse
	for _, provider := range providerarr {
		switch provider {
		case "aws":
			cred := stsmodule.CspCredential{
				Provider: "aws",
			}
			securityToken, err := awsStsModule.GetAwsSecurityToken(c)
			if err != nil {
				cred.Credential = err.Error()
			} else {
				cred.Credential = securityToken.AssumeRoleWithWebIdentityResult.Credentials
			}
			mciamCspCredentialsResponse.CspCredentials = append(mciamCspCredentialsResponse.CspCredentials, cred)
		case "alibaba":
			cred := stsmodule.CspCredential{
				Provider: "alibaba",
			}
			securityToken, err := alibabaStsModule.GetAlibabaSecurityToken(c)
			if err != nil {
				cred.Credential = err.Error()
			} else {
				cred.Credential = securityToken.Credentials
			}
			mciamCspCredentialsResponse.CspCredentials = append(mciamCspCredentialsResponse.CspCredentials, cred)
		default:
			mciamCspCredentialsResponse.UnSupportedProviders = append(mciamCspCredentialsResponse.UnSupportedProviders, provider)
		}
	}
	return c.Render(http.StatusOK, r.JSON(mciamCspCredentialsResponse))
}
