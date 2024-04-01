package actions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mc_iam_manager/models"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gobuffalo/buffalo"
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
	// 	ID       string `json:"id" db:"id"`
	// 	UserId string `json:"UserId" db:"UserId"`
	// 	Password string `json:"pasword"`
	var user models.UserEntity
	if err := c.Bind(&user); err != nil {
		return c.Render(http.StatusServiceUnavailable,
			r.JSON(map[string]string{"error": err.Error()}))
	}

	formData := url.Values{
		"username":      {user.UserId},
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

	respBody, err := ioutil.ReadAll(resp.Body)
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
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "AuthLogoutHandler"}))
}

func AuthGetSecurityKeyHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "AuthGetSecurityKeyHandler"}))
}
