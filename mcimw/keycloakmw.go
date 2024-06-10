package mcimw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mc_iam_manager/iammodels"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
)

var (
	//IDP use bool
	KEYCLOAK_HOST string
	KEYCLOAK      *gocloak.GoCloak

	KEYCLAOK_REALM         string
	KEYCLAOK_CLIENT        string
	KEYCLAOK_CLIENT_SECRET string
)

func init() {
	//default set of console KEYCLOAK
	KEYCLOAK_HOST = envy.Get("KEYCLOAK_HOST", "")
	KEYCLOAK = gocloak.NewClient(KEYCLOAK_HOST)
	KEYCLAOK_REALM = envy.Get("KEYCLAOK_REALM", "mciam")
	KEYCLAOK_CLIENT = envy.Get("KEYCLAOK_CLIENT", "mciammanager")
	KEYCLAOK_CLIENT_SECRET = envy.Get("KEYCLAOK_CLIENT_SECRET", "mciammanagerclientsecret")
}

func BeginKeycloakAuth(res http.ResponseWriter, req *http.Request) {
	reqUri := req.RequestURI
	reqMethod := req.Method
	res.Header().Set("Content-Type", "application/json")
	if strings.HasSuffix(reqUri, "/login") && reqMethod == "POST" {
		keycloakauthLoginHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/login/refresh") && reqMethod == "POST" {
		keycloakauthLoginRefreshHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/logout") && reqMethod == "POST" {
		keycloakauthLogoutHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/userinfo") && reqMethod == "GET" {
		keycloakauthGetUserInfo(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/validate") && reqMethod == "GET" {
		keycloakauthGetUserValidate(res, req)
		return
	}
	res.WriteHeader(http.StatusBadRequest)
	err := errors.New("NO MATCH AUTH")
	fmt.Fprintln(res, err.Error())
}

func keycloakauthLoginHandler(res http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	user := &iammodels.UserLogin{}
	err := decoder.Decode(&user)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}
	defer req.Body.Close()

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.Id, Name: "id"},
		&validators.StringIsPresent{Field: user.Password, Name: "password"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, validateErr.Error())
		return
	}

	ctx := context.Background()
	accessTokenResponse, err := KEYCLOAK.Login(ctx, KEYCLAOK_CLIENT, KEYCLAOK_CLIENT_SECRET, KEYCLAOK_REALM, user.Id, user.Password)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}

	jsonData, err := json.Marshal(accessTokenResponse)
	if err != nil {
		fmt.Println("JSON 변환 오류:", err)
		return
	}

	res.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(res, string(jsonData))
}

func keycloakauthLoginRefreshHandler(res http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	user := &iammodels.UserLoginRefresh{}
	err := decoder.Decode(&user)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}
	defer req.Body.Close()

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, validateErr.Error())
		return
	}

	ctx := context.Background()
	accessTokenResponse, err := KEYCLOAK.RefreshToken(ctx, user.RefreshToken, KEYCLAOK_CLIENT, KEYCLAOK_CLIENT_SECRET, KEYCLAOK_REALM)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}

	jsonData, err := json.Marshal(accessTokenResponse)
	if err != nil {
		fmt.Println("JSON 변환 오류:", err)
		return
	}

	res.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(res, string(jsonData))
}

func keycloakauthLogoutHandler(res http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	user := &iammodels.UserLogout{}
	err := decoder.Decode(&user)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}
	defer req.Body.Close()

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, validateErr.Error())
		return
	}

	ctx := context.Background()
	err = KEYCLOAK.Logout(ctx, KEYCLAOK_CLIENT, KEYCLAOK_CLIENT_SECRET, KEYCLAOK_REALM, user.RefreshToken)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
	fmt.Fprintln(res, nil)
}

func keycloakauthGetUserInfo(res http.ResponseWriter, req *http.Request) {
	accessToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, validateErr.Error())
		return
	}

	ctx := context.Background()

	userinfo, err := KEYCLOAK.GetUserInfo(ctx, accessToken, KEYCLAOK_REALM)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}

	jsonData, err := json.Marshal(userinfo)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
	fmt.Fprintln(res, string(jsonData))
}

func keycloakauthGetUserValidate(res http.ResponseWriter, req *http.Request) {
	accessToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: accessToken, Name: "Authorization"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, validateErr.Error())
		return
	}

	ctx := context.Background()

	_, err := KEYCLOAK.GetUserInfo(ctx, accessToken, KEYCLAOK_REALM)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
	fmt.Fprintln(res, nil)
}
