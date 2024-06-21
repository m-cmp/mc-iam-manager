package actions

import (
	"log"
	"mc_iam_manager/handler/securitykeyprovider"
	"mc_iam_manager/handler/securitykeyprovider/alibaba"
	"mc_iam_manager/handler/securitykeyprovider/aws"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
)

func AuthSecuritykeyProviderHandler(c buffalo.Context) error {
	providers := c.Param("providers")
	var providerarr []string
	if providers != "" {
		providerarr = strings.Split(providers, ",")
	} else {
		providerarr = []string{"aws", "alibaba"}
	}
	var mciamCspCredentialsResponse securitykeyprovider.MciamCspCredentialsResponse
	for _, provider := range providerarr {
		switch provider {
		case "aws":
			t := aws.AWS{}
			res, err := securitykeyprovider.GetKey(c, t)
			if err != nil {
				log.Println(err.Error())
				return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error : aws :": err.Error()}))
			}
			mciamCspCredentialsResponse.CspCredentials = append(mciamCspCredentialsResponse.CspCredentials, *res)
		case "alibaba":
			t := alibaba.Alibaba{}
			res, err := securitykeyprovider.GetKey(c, t)
			if err != nil {
				log.Println(err.Error())
				return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error : alibaba :": err.Error()}))
			}
			mciamCspCredentialsResponse.CspCredentials = append(mciamCspCredentialsResponse.CspCredentials, *res)
		default:

		}
	}
	return c.Render(http.StatusOK, r.JSON(mciamCspCredentialsResponse))
}
