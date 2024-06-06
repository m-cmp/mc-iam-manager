package actions

import (
	"fmt"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/stsmodule"
	alibabaStsModule "mc_iam_manager/stsmodule/alibaba"
	awsStsModule "mc_iam_manager/stsmodule/aws"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
)

var (
	//default set of console Admin var
	ADMINUSERID       string
	ADMINUSERPASSWORD string

	//IDP use bool
	KEYCLOAK_HOST           string
	KEYCLOAK                *gocloak.GoCloak
	KEYCLAOK_ADMIN          string
	KEYCLAOK_ADMIN_PASSWORD string
	KEYCLAOK_REALM          string
	KEYCLAOK_CLIENT         string
	KEYCLAOK_CLIENT_SECRET  string
)

func init() {
	//default set of console Admin var
	ADMINUSERID = envy.Get("ADMINUSERID", "mcpuser")
	ADMINUSERPASSWORD = envy.Get("ADMINUSERPASSWORD", "mcpuserpassword")

	//default set of console KEYCLOAK
	KEYCLOAK_HOST = envy.Get("KEYCLOAK_HOST", "")
	KEYCLOAK = gocloak.NewClient(KEYCLOAK_HOST)
	KEYCLAOK_REALM = envy.Get("KEYCLAOK_REALM", "mciam")
	KEYCLAOK_CLIENT = envy.Get("KEYCLAOK_CLIENT", "mciammanager")
	KEYCLAOK_CLIENT_SECRET = envy.Get("KEYCLAOK_CLIENT_SECRET", "mciammanagerclientsecret")
	KEYCLAOK_ADMIN = envy.Get("KEYCLAOK_ADMIN", "admin")
	KEYCLAOK_ADMIN_PASSWORD = envy.Get("KEYCLAOK_ADMIN_PASSWORD", "admin")
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

	accessTokenResponse, err := KEYCLOAK.Login(c, KEYCLAOK_CLIENT, KEYCLAOK_CLIENT_SECRET, KEYCLAOK_REALM, user.Id, user.Password)
	if err != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
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
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	accessTokenResponse, err := KEYCLOAK.RefreshToken(c, user.RefreshToken, KEYCLAOK_CLIENT, KEYCLAOK_CLIENT_SECRET, KEYCLAOK_REALM)
	if err != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
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

	err := KEYCLOAK.Logout(c, KEYCLAOK_CLIENT, KEYCLAOK_CLIENT_SECRET, KEYCLAOK_REALM, user.RefreshToken)
	if err != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	return c.Render(http.StatusOK, nil)
}

func AuthGetUserValidate(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	_, err := KEYCLOAK.GetUserInfo(c, accessToken, KEYCLAOK_REALM)
	if err != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	return c.Render(http.StatusOK, nil)
}

func AuthGetUserInfo(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": validateErr.Error()}))
	}

	userinfo, err := KEYCLOAK.GetUserInfo(c, accessToken, KEYCLAOK_REALM)
	if err != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(userinfo))
}

func AuthGetSecurityKeyHandler(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

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
