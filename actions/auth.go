package actions

import (
	"fmt"
	"mc_iam_manager/stsmodule"
	alibabaStsModule "mc_iam_manager/stsmodule/alibaba"
	awsStsModule "mc_iam_manager/stsmodule/aws"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
)

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
