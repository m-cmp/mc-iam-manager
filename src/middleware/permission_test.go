package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// callPlatformRoleMiddleware 는 platformRoles 를 컨텍스트에 넣고(setRoles=false 면 넣지 않음)
// PlatformRoleMiddleware(level) 을 통과하면 200, 거부되면 HTTPError 의 코드를 돌려준다.
func callPlatformRoleMiddleware(t *testing.T, level PermissionLevel, platformRoles []string, setRoles bool) int {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/company", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if setRoles {
		c.Set("platformRoles", platformRoles)
	}

	next := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	err := PlatformRoleMiddleware(level)(next)(c)
	if err == nil {
		return rec.Code
	}
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok, "unexpected error type: %T", err)
	return he.Code
}

func TestPlatformRoleMiddleware_Read(t *testing.T) {
	// 역할 없는 사용자(가입 승인 직후) — IAM-BUG-038
	require.Equal(t, http.StatusForbidden, callPlatformRoleMiddleware(t, Read, []string{}, true))
	require.Equal(t, http.StatusForbidden, callPlatformRoleMiddleware(t, Read, nil, true))
	// 역할 정보 자체가 없으면 기존대로 401
	require.Equal(t, http.StatusUnauthorized, callPlatformRoleMiddleware(t, Read, nil, false))

	for _, roles := range [][]string{{"viewer"}, {"operator"}, {"admin"}, {"platformAdmin"}} {
		require.Equal(t, http.StatusOK, callPlatformRoleMiddleware(t, Read, roles, true), "roles=%v", roles)
	}
}

func TestPlatformRoleMiddleware_WriteManageUnchanged(t *testing.T) {
	require.Equal(t, http.StatusForbidden, callPlatformRoleMiddleware(t, Write, []string{}, true))
	require.Equal(t, http.StatusForbidden, callPlatformRoleMiddleware(t, Write, []string{"viewer"}, true))
	require.Equal(t, http.StatusOK, callPlatformRoleMiddleware(t, Write, []string{"admin"}, true))

	require.Equal(t, http.StatusForbidden, callPlatformRoleMiddleware(t, Manage, []string{"admin"}, true))
	require.Equal(t, http.StatusOK, callPlatformRoleMiddleware(t, Manage, []string{"platformAdmin"}, true))
}
