package mcimw

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gobuffalo/envy"
)

type mcmpauth struct {
	proxy *httputil.ReverseProxy
}

var (
	mciamHostUrl, _ = url.Parse(envy.Get("MCIAM_HOST", "http://localhost:5000"))
	EnvMcmpauth     = mcmpauth{
		proxy: httputil.NewSingleHostReverseProxy(mciamHostUrl),
	}
)

func (m mcmpauth) AuthLoginHandler(res http.ResponseWriter, req *http.Request) {
	m.proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthLoginRefreshHandler(res http.ResponseWriter, req *http.Request) {
	m.proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthLogoutHandler(res http.ResponseWriter, req *http.Request) {
	m.proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthGetUserInfo(res http.ResponseWriter, req *http.Request) {
	m.proxy.ServeHTTP(res, req)
}

func (m mcmpauth) AuthGetUserValidate(res http.ResponseWriter, req *http.Request) {
	m.proxy.ServeHTTP(res, req)
}
