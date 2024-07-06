package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

type mciamAuth interface {
	AuthLoginHandler(http.ResponseWriter, *http.Request)
	AuthLoginRefreshHandler(http.ResponseWriter, *http.Request)
	AuthLogoutHandler(http.ResponseWriter, *http.Request)
	AuthGetUserInfo(http.ResponseWriter, *http.Request)
	AuthGetUserValidate(http.ResponseWriter, *http.Request)
	AuthGetCerts(http.ResponseWriter, *http.Request)
}

var (
	AuthMethod mciamAuth
)

func BeginAuthHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	reqUri := req.RequestURI
	reqMethod := req.Method

	if strings.HasSuffix(reqUri, "/login") && reqMethod == "POST" {
		AuthMethod.AuthLoginHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/login/refresh") && reqMethod == "POST" {
		AuthMethod.AuthLoginRefreshHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/logout") && reqMethod == "POST" {
		AuthMethod.AuthLogoutHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/userinfo") && reqMethod == "GET" {
		AuthMethod.AuthGetUserInfo(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/validate") && reqMethod == "GET" {
		AuthMethod.AuthGetUserValidate(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/certs") && reqMethod == "GET" {
		AuthMethod.AuthGetCerts(res, req)
		return
	}

	res.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(res).Encode(map[string]interface{}{
		"error": http.StatusText(http.StatusBadRequest),
	})
}
