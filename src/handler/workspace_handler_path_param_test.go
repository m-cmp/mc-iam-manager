package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 사용자 alice 는 ws1, bob 은 ws2 에만 역할이 있다.
func setupWorkspaceMembersTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	for _, ddl := range []string{
		`CREATE TABLE mcmp_users (id INTEGER PRIMARY KEY, username TEXT)`,
		`CREATE TABLE mcmp_workspaces (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE mcmp_role_masters (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE mcmp_user_workspace_roles (user_id INTEGER, workspace_id INTEGER, role_id INTEGER, created_at DATETIME,
			PRIMARY KEY (user_id, workspace_id, role_id))`,
		`INSERT INTO mcmp_users (id, username) VALUES (1, 'alice'), (2, 'bob')`,
		`INSERT INTO mcmp_workspaces (id, name) VALUES (1, 'ws1'), (2, 'ws2')`,
		`INSERT INTO mcmp_role_masters (id, name) VALUES (10, 'viewer')`,
		`INSERT INTO mcmp_user_workspace_roles (user_id, workspace_id, role_id, created_at) VALUES
			(1, 1, 10, CURRENT_TIMESTAMP), (2, 2, 10, CURRENT_TIMESTAMP)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	return db
}

func callListUsersAndRolesByWorkspaces(t *testing.T, h *WorkspaceHandler, workspaceID, body string) []model.UserWorkspaceRole {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/id/"+workspaceID+"/users/list", strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/workspaces/id/:workspaceId/users/list")
	c.SetParamNames("workspaceId")
	c.SetParamValues(workspaceID)

	require.NoError(t, h.ListUsersAndRolesByWorkspaces(c))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got []model.UserWorkspaceRole
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

func usernames(rows []model.UserWorkspaceRole) []string {
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Username)
	}
	return names
}

// IAM-BUG-049: path 의 workspaceId 를 읽지 않아 모든 워크스페이스의 멤버가 반환됐다.
func TestListUsersAndRolesByWorkspaces_FiltersByPathWorkspaceID(t *testing.T) {
	h := NewWorkspaceHandler(setupWorkspaceMembersTestDB(t))

	// 콘솔은 body 없이 path 로만 보낸다
	got := callListUsersAndRolesByWorkspaces(t, h, "2", "")
	require.Equal(t, []string{"bob"}, usernames(got), "ws2 멤버 목록에 ws1 에만 역할이 있는 alice 가 있으면 안 된다")
	require.Equal(t, uint(2), got[0].WorkspaceID)

	got = callListUsersAndRolesByWorkspaces(t, h, "1", "")
	require.Equal(t, []string{"alice"}, usernames(got))
}

// path 값이 body 의 workspaceId 보다 우선한다.
func TestListUsersAndRolesByWorkspaces_PathOverridesBody(t *testing.T) {
	h := NewWorkspaceHandler(setupWorkspaceMembersTestDB(t))

	got := callListUsersAndRolesByWorkspaces(t, h, "2", `{"workspaceId":"1"}`)
	require.Equal(t, []string{"bob"}, usernames(got))
}

// 같은 패턴: POST /api/workspaces/id/:id/users 는 path 의 워크스페이스를 무시하고 body 값만 썼다.
func TestAddUserToWorkspace_UsesPathWorkspaceID(t *testing.T) {
	h := NewWorkspaceHandler(setupWorkspaceMembersTestDB(t))

	// 이전에는 body 값만 써서 각각 400 이 났다. path 값으로 처리되면 워크스페이스 ID 검증 오류가 없어야 한다.
	for name, body := range map[string]string{
		"body 에 workspaceId 없음":   `{"userId":"1"}`,
		"body workspaceId 가 잘못됨": `{"userId":"1","workspaceId":"not-a-number"}`,
	} {
		t.Run(name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/workspaces/id/2/users", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/workspaces/id/:id/users")
			c.SetParamNames("id")
			c.SetParamValues("2")

			require.NoError(t, h.AddUserToWorkspace(c))
			require.NotContains(t, rec.Body.String(), "Workspace ID and User ID are required")
			require.NotContains(t, rec.Body.String(), "Invalid workspace ID format")
		})
	}
}
