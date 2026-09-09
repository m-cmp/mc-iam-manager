package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// IAM-TECH-009 (B): actionName이 비면 UMA 스코프가 빈 문자열이 되어 리소스만 맞으면
// 통과하던 경로를 막는다. 검증은 RPT 디코드보다 앞서므로 Keycloak 없이 400을 확인할 수 있다.
func TestMcmpApiCall_MissingActionName_ReturnsBadRequest(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	h := newMcmpApiCallTestHandler(setupMcmpApiCallTestDB(t))

	for _, body := range []string{
		`{"serviceName":"mc-infra-manager","actionName":""}`,
		`{"serviceName":"","actionName":"GetAllNs"}`,
		`{"serviceName":"mc-infra-manager"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/mcmp-apis/call", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer dummy")
		rec := httptest.NewRecorder()

		require.NoError(t, h.McmpApiCall(e.NewContext(req, rec)))
		require.Equal(t, http.StatusBadRequest, rec.Code, body)
		require.Contains(t, rec.Body.String(), "serviceName and actionName are required", body)
	}
}
