package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupListUserMenuTestDB 메뉴 4개(상위 1 + 하위 3)와 플랫폼 역할 admin·viewer·empty 를 만든다.
// admin=전체, viewer=dashboard 만, empty=매핑 없음.
func setupListUserMenuTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Menu{}, &model.RoleMaster{}, &model.RoleSub{}, &model.RoleMenuMapping{}))

	menus := []model.Menu{
		{ID: "operations", DisplayName: "Operations", ResType: "menu", Priority: 1, MenuNumber: 1},
		{ID: "dashboard", ParentID: "operations", DisplayName: "Dashboard", ResType: "menu", Priority: 1, MenuNumber: 2},
		{ID: "approvals", ParentID: "operations", DisplayName: "Approvals", ResType: "menu", Priority: 1, MenuNumber: 3},
		{ID: "users", ParentID: "operations", DisplayName: "Users", ResType: "menu", Priority: 1, MenuNumber: 4},
	}
	for i := range menus {
		require.NoError(t, db.Create(&menus[i]).Error)
	}

	mapping := map[string][]string{
		"admin":  {"operations", "dashboard", "approvals", "users"},
		"viewer": {"dashboard"},
		"empty":  {},
	}
	for _, name := range []string{"admin", "viewer", "empty"} {
		role := model.RoleMaster{Name: name}
		require.NoError(t, db.Create(&role).Error)
		require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)
		for _, menuID := range mapping[name] {
			require.NoError(t, db.Create(&model.RoleMenuMapping{RoleID: role.ID, MenuID: menuID}).Error)
		}
	}
	return db
}

func callListUserMenu(t *testing.T, h *MenuHandler, platformRoles []string) (string, []string) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/users/menus/list", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("platformRoles", platformRoles)

	require.NoError(t, h.ListUserMenu(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var menus []model.Menu
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &menus))
	ids := make([]string, 0, len(menus))
	for _, m := range menus {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	return strings.TrimSpace(rec.Body.String()), ids
}

func callListUserMenuTree(t *testing.T, h *MenuHandler, platformRoles []string) string {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/users/menus-tree/list", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("platformRoles", platformRoles)

	require.NoError(t, h.ListUserMenuTree(c))
	require.Equal(t, http.StatusOK, rec.Code)
	return strings.TrimSpace(rec.Body.String())
}

// TC-IAM-BUG-037-01: 플랫폼 역할이 없는 사용자(가입 승인 직후)는 빈 메뉴 [] (null 아님)
func TestListUserMenu_NoPlatformRole_ReturnsEmpty(t *testing.T) {
	h := NewMenuHandler(setupListUserMenuTestDB(t))

	body, ids := callListUserMenu(t, h, nil)
	require.Equal(t, "[]", body)
	require.Empty(t, ids)

	require.Equal(t, "[]", callListUserMenuTree(t, h, nil))
}

// TC-IAM-BUG-037-02: DB 에 없는 임의 역할만 가진 사용자는 admin fallback 을 타지 않는다
func TestListUserMenu_UnknownRole_NoAdminFallback(t *testing.T) {
	h := NewMenuHandler(setupListUserMenuTestDB(t))

	body, _ := callListUserMenu(t, h, []string{"foo"})
	require.Equal(t, "[]", body)
	require.Equal(t, "[]", callListUserMenuTree(t, h, []string{"foo"}))
}

// TC-IAM-BUG-037-03: platformAdmin(DB 역할 아님)은 admin 메뉴로 fallback — 기존 동작 유지
func TestListUserMenu_PlatformAdmin_FallsBackToAdmin(t *testing.T) {
	h := NewMenuHandler(setupListUserMenuTestDB(t))

	_, ids := callListUserMenu(t, h, []string{"platformAdmin"})
	require.Equal(t, []string{"approvals", "dashboard", "operations", "users"}, ids)

	tree := callListUserMenuTree(t, h, []string{"platformAdmin"})
	require.Contains(t, tree, `"approvals"`)
	require.Contains(t, tree, `"users"`)
}

// TC-IAM-BUG-037-04: DB 역할 사용자는 매핑된 메뉴(+상위 메뉴)만 — 기존 동작 유지
func TestListUserMenu_SeedRoles_Unchanged(t *testing.T) {
	h := NewMenuHandler(setupListUserMenuTestDB(t))

	_, ids := callListUserMenu(t, h, []string{"admin"})
	require.Equal(t, []string{"approvals", "dashboard", "operations", "users"}, ids)

	_, ids = callListUserMenu(t, h, []string{"viewer"})
	require.Equal(t, []string{"dashboard", "operations"}, ids)

	tree := callListUserMenuTree(t, h, []string{"viewer"})
	require.Contains(t, tree, `"dashboard"`)
	require.NotContains(t, tree, `"approvals"`)
}

// TC-IAM-BUG-037-05: 매핑이 하나도 없는 DB 역할은 빈 메뉴 (빈 MenuIDs → 전체 메뉴 방지)
func TestListUserMenu_RoleWithoutMappings_ReturnsEmpty(t *testing.T) {
	h := NewMenuHandler(setupListUserMenuTestDB(t))

	body, _ := callListUserMenu(t, h, []string{"empty"})
	require.Equal(t, "[]", body)
	require.Equal(t, "[]", callListUserMenuTree(t, h, []string{"empty"}))
}
