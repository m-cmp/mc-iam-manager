package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// newSwaggerTestServer 는 main() 과 같은 방식으로 인증 미들웨어와 Swagger 라우트를 등록한다.
func newSwaggerTestServer(swaggerEnabled bool) *echo.Echo {
	e := echo.New()
	e.Use(authSkipMiddleware([]string{"/readyz"}, swaggerEnabled))
	e.GET("/readyz", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	e.GET("/api/users/me", func(c echo.Context) error { return c.String(http.StatusOK, "me") })
	e.GET("/api/foo/swagger/index.html", func(c echo.Context) error { return c.String(http.StatusOK, "leak") })
	if swaggerEnabled {
		e.GET("/swagger/*", echoSwagger.WrapHandler)
	}
	return e
}

func TestSwaggerAuthSkip(t *testing.T) {
	cases := []struct {
		name           string
		swaggerEnabled bool
		path           string
		want           int
	}{
		{"enabled: swagger UI without token", true, "/swagger/index.html", http.StatusOK},
		{"enabled: swagger spec without token", true, "/swagger/doc.json", http.StatusOK},
		{"enabled: API still requires token", true, "/api/users/me", http.StatusUnauthorized},
		{"enabled: swagger suffix elsewhere is not exempt", true, "/api/foo/swagger/index.html", http.StatusUnauthorized},
		{"enabled: existing skip path still open", true, "/readyz", http.StatusOK},
		{"disabled: swagger UI requires token", false, "/swagger/index.html", http.StatusUnauthorized},
		{"disabled: swagger spec requires token", false, "/swagger/doc.json", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newSwaggerTestServer(tc.swaggerEnabled)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != tc.want {
				t.Fatalf("GET %s (swaggerEnabled=%v): got %d, want %d", tc.path, tc.swaggerEnabled, rec.Code, tc.want)
			}
		})
	}
}
