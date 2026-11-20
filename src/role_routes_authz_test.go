package main

import (
	"os"
	"regexp"
	"testing"
)

// roles 그룹 라우트 권한 표 (IAM-BUG-054)
//
// 미들웨어 없이 열려 있던 역할 쓰기 라우트(CspRole·csp-roles master·platform-roles·workspace-roles)는
// 로그인만 하면 누구나 호출할 수 있었다. main.go 의 실제 라우트 등록을 진실로 삼아 라우트별 권한을 고정한다.

var reRolesRoute = regexp.MustCompile(`(?m)^\s*roles\.(GET|POST|PUT|DELETE)\("([^"]*)",\s*roleHandler\.\w+(.*)\)`)

const (
	authzNone          = "none"
	authzRead          = "middleware.PlatformRoleMiddleware(middleware.Read)"
	authzWrite         = "middleware.PlatformRoleMiddleware(middleware.Write)"
	authzManage        = "middleware.PlatformRoleMiddleware(middleware.Manage)"
	authzPlatformAdmin = "middleware.PlatformAdminMiddleware"
)

func loadRolesRouteAuthz(t *testing.T) map[string]string {
	t.Helper()
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	routes := map[string]string{}
	for _, m := range reRolesRoute.FindAllStringSubmatch(string(src), -1) {
		key := m[1] + " " + m[2]
		mw := authzNone
		for _, cand := range []string{authzRead, authzWrite, authzManage, authzPlatformAdmin} {
			if regexp.MustCompile(regexp.QuoteMeta(cand)).MatchString(m[3]) {
				mw = cand
			}
		}
		routes[key] = mw
	}
	return routes
}

func TestRolesRouteAuthz(t *testing.T) {
	routes := loadRolesRouteAuthz(t)
	want := map[string]string{
		// CspRole: AWS 대상 CSP Role 생성·수정·삭제를 일으키므로 PlatformAdmin (/csp-policies 쓰기와 같은 기준)
		"POST /csp":               authzPlatformAdmin,
		"POST /csp/batch":         authzPlatformAdmin,
		"PUT /csp/id/:roleId":     authzPlatformAdmin,
		"DELETE /csp/id/:roleId":  authzPlatformAdmin,
		"POST /csp/list":          authzRead,
		"GET /csp/id/:roleId":     authzRead,
		"GET /csp/name/:roleName": authzRead,

		// csp-roles master: csp Sub 를 가진 Master Role CRUD 라 Master Role 생성·수정(POST/PUT /roles)과 같은 기준
		"POST /csp-roles/master":              authzWrite,
		"PUT /csp-roles/master/id/:roleId":    authzWrite,
		"DELETE /csp-roles/master/id/:roleId": authzWrite,
		"POST /csp-roles":                     authzManage,
		"DELETE /csp-roles":                   authzManage,
		"POST /csp-roles/list":                authzRead,
		// 웹 콘솔 임시자격증명 화면이 플랫폼 역할 없는 워크스페이스 사용자에 대해서도 호출한다(발급 가능 CSP 판정)
		"GET /csp-roles/id/:roleId": authzNone,

		// platform-roles · workspace-roles: Master Role 생성·수정(POST/PUT /roles)과 같은 기준
		"POST /platform-roles":                authzWrite,
		"PUT /platform-roles/id/:roleId":      authzWrite,
		"DELETE /platform-roles/id/:roleId":   authzWrite,
		"POST /platform-roles/list":           authzRead,
		"GET /platform-roles/id/:roleId":      authzRead,
		"GET /platform-roles/name/:roleName":  authzRead,
		"POST /workspace-roles":               authzWrite,
		"PUT /workspace-roles/id/:roleId":     authzWrite,
		"DELETE /workspace-roles/id/:roleId":  authzWrite,
		"POST /workspace-roles/list":          authzRead,
		"GET /workspace-roles/id/:roleId":     authzRead,
		"GET /workspace-roles/name/:roleName": authzRead,

		// 기존 Master Role 쓰기 (변경 없음)
		"POST ":              authzWrite,
		"PUT /id/:roleId":    authzWrite,
		"DELETE /id/:roleId": authzWrite,
	}
	for route, mw := range want {
		got, ok := routes[route]
		if !ok {
			t.Errorf("route %q not found in main.go roles group", route)
			continue
		}
		if got != mw {
			t.Errorf("route %q: middleware = %s, want %s", route, got, mw)
		}
	}
}

// 쓰기 메서드(POST/PUT/DELETE)인데 미들웨어가 없는 roles 라우트가 새로 생기지 않도록 막는다.
// 조회 성격의 POST(.../list)와 의도적으로 열어 둔 라우트만 예외로 둔다.
func TestRolesWriteRoutesHaveAuthz(t *testing.T) {
	allowedOpen := map[string]bool{
		"POST /list":          true, // Master Role 목록 조회(기존)
		"POST /mappings/list": true,
		"POST /mappings/platform-roles/users/list":  true,
		"POST /mappings/workspace-roles/users/list": true,
		"POST /mappings/csp-roles/list":             true,
	}
	reMethod := regexp.MustCompile(`^(POST|PUT|DELETE) `)
	for route, mw := range loadRolesRouteAuthz(t) {
		if !reMethod.MatchString(route) || mw != authzNone || allowedOpen[route] {
			continue
		}
		t.Errorf("write route %q has no authorization middleware", route)
	}
}
