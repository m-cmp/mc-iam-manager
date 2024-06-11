package mcimw

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gobuffalo/envy"
)

type mcmpauth struct{}

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

func (m mcmpauth) AuthLoginHandler(res http.ResponseWriter, req *http.Request) {
	proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthLoginRefreshHandler(res http.ResponseWriter, req *http.Request) {
	proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthLogoutHandler(res http.ResponseWriter, req *http.Request) {
	proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthGetUserInfo(res http.ResponseWriter, req *http.Request) {
	proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthGetUserValidate(res http.ResponseWriter, req *http.Request) {
	proxy.ServeHTTP(res, req)
}
