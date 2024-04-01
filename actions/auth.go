package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
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
	user.Id = c.Request().Form.Get("id")
	user.Password = c.Request().Form.Get("password")
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

	req, err := http.NewRequest("POST", tokenEndpoint.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		fmt.Println("Failed to create request:", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"code": resp.Status}))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": err.Error()}))
	}

	var accessTokenResponse models.KeycloakAccessTokenResponse
	jsonerr := json.Unmarshal(respBody, &accessTokenResponse)
	if jsonerr != nil {
		fmt.Println("Failed to parse response:", err)
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"err": jsonerr.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(accessTokenResponse))
}

func AuthLogoutHandler(c buffalo.Context) error {

	accessToken := c.Request().Header.Get("Authorization")
	fmt.Println("********************************accessToken", accessToken)
	refreshToken := c.Request().FormValue("refresh_token")

	formData := url.Values{
		"client_id":     {keycloakClient},
		"client_secret": {keycloakClientSecret},
		"refresh_token": {refreshToken},
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
	req.Header.Set("Authorization", accessToken)

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

	fmt.Println("********************************resp.Status", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("********************************respBody", string(respBody))

	return c.Render(http.StatusNoContent, r.JSON(""))
}

func AuthGetSecurityKeyHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "AuthGetSecurityKeyHandler"}))
}
