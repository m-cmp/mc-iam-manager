package mcimw

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gobuffalo/envy"
)

var (
	MCIAM_HOST string
	proxy      *httputil.ReverseProxy
)

func init() {
	MCIAM_HOST = envy.Get("MCIAM_HOST", "http://localhost:5000")
	mciamHostUrl, err := url.Parse(MCIAM_HOST)
	if err != nil {
		panic(err)
	}
	proxy = httputil.NewSingleHostReverseProxy(mciamHostUrl)
}

func BeginMCIAMAuth(res http.ResponseWriter, req *http.Request) {
	reqUri := req.RequestURI
	reqMethod := req.Method
	if strings.HasSuffix(reqUri, "/login") && reqMethod == "POST" {
		proxy.ServeHTTP(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/login/refresh") && reqMethod == "POST" {
		proxy.ServeHTTP(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/logout") && reqMethod == "POST" {
		proxy.ServeHTTP(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/userinfo") && reqMethod == "GET" {
		proxy.ServeHTTP(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/validate") && reqMethod == "GET" {
		proxy.ServeHTTP(res, req)
		return
	}
	res.WriteHeader(http.StatusBadRequest)
	err := errors.New("NO MATCH AUTH")
	fmt.Fprintln(res, err.Error())
}
