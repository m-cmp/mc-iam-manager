package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// IAM-TECH-003 방안2: /api/mcmp-apis/call의 nsId 소유권 기반 세밀 권한 제어.
//
// mcmp_user_workspace_roles는 raw SQL로 직접 만든다 (AutoMigrate(&model.User{})를
// 함께 돌리면 User의 many2many 태그가 같은 테이블 이름을 2컬럼짜리 role 매핑용으로
// 재정의해버려 충돌한다 - role_handler_deleterole_realmrole_test.go의 setupDeleteRoleTestDB
// 워크어라운드와 동일한 이유). authorizeMcmpApiAction의 kcUserID→localUserID 변환은
// Keycloak 동기화를 유발할 수 있어 여기서는 검증하지 않고, 그 이전 분기와
// authorizeNsOwnership(로컬 userID를 직접 받는 분리된 함수)만 테스트한다.
func setupMcmpApiCallTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Workspace{},
		&model.Project{},
		&mcmpapi.McmpApiAction{},
	))
	require.NoError(t, db.Exec(`CREATE TABLE mcmp_user_workspace_roles (
		user_id integer, workspace_id integer, role_id integer,
		username text, workspace_name text, role_name text, created_at datetime,
		PRIMARY KEY (user_id, workspace_id, role_id)
	)`).Error)
	return db
}

func newMcmpApiCallTestHandler(db *gorm.DB) *McmpApiHandler {
	return &McmpApiHandler{
		projectService:   service.NewProjectService(db),
		workspaceService: service.NewWorkspaceService(db),
		mcmpApiRepo:      repository.NewMcmpApiRepository(db),
	}
}

func TestAuthorizeMcmpApiAction_PlatformAdmin_Passes(t *testing.T) {
	h := newMcmpApiCallTestHandler(setupMcmpApiCallTestDB(t))
	req := &model.McmpApiCallRequest{ServiceName: "mc-infra-manager", ActionName: "DelAllNs"}

	status, msg := h.authorizeMcmpApiAction(context.Background(), true, "", req)

	require.Equal(t, 0, status, "platformAdmin은 액션/nsId 조회 없이 통과해야 한다: %s", msg)
}

func TestAuthorizeMcmpApiAction_UndefinedAction_ReturnsForbidden(t *testing.T) {
	h := newMcmpApiCallTestHandler(setupMcmpApiCallTestDB(t))
	req := &model.McmpApiCallRequest{ServiceName: "mc-infra-manager", ActionName: "NoSuchAction"}

	status, _ := h.authorizeMcmpApiAction(context.Background(), false, "kc-user-1", req)

	require.Equal(t, http.StatusForbidden, status, "service-actions.yaml에 정의되지 않은 액션은 거부해야 한다")
}

func TestAuthorizeMcmpApiAction_ListStyleAction_NonAdmin_ReturnsForbidden(t *testing.T) {
	db := setupMcmpApiCallTestDB(t)
	require.NoError(t, db.Create(&mcmpapi.McmpApiAction{
		ServiceName: "mc-infra-manager", ActionName: "GetAllNs", Method: "get", ResourcePath: "/ns",
	}).Error)
	h := newMcmpApiCallTestHandler(db)
	req := &model.McmpApiCallRequest{ServiceName: "mc-infra-manager", ActionName: "GetAllNs"}

	status, _ := h.authorizeMcmpApiAction(context.Background(), false, "kc-user-1", req)

	require.Equal(t, http.StatusForbidden, status, "path param(nsId)이 없는 목록류 액션은 non-admin에게 계속 403이어야 한다")
}

func TestAuthorizeMcmpApiAction_NsScopedAction_MissingNsIdPathParam_ReturnsBadRequest(t *testing.T) {
	db := setupMcmpApiCallTestDB(t)
	require.NoError(t, db.Create(&mcmpapi.McmpApiAction{
		ServiceName: "mc-infra-manager", ActionName: "GetAllInfra", Method: "get", ResourcePath: "/ns/{nsId}/infra",
	}).Error)
	h := newMcmpApiCallTestHandler(db)
	req := &model.McmpApiCallRequest{
		ServiceName: "mc-infra-manager", ActionName: "GetAllInfra",
		RequestParams: model.McmpApiRequestParams{PathParams: map[string]string{}},
	}

	status, _ := h.authorizeMcmpApiAction(context.Background(), false, "kc-user-1", req)

	require.Equal(t, http.StatusBadRequest, status, "resourcePath가 {nsId}를 요구하는데 pathParams에 없으면 400이어야 한다")
}

func TestAuthorizeMcmpApiAction_NsScopedAction_NoAuthContext_ReturnsUnauthorized(t *testing.T) {
	db := setupMcmpApiCallTestDB(t)
	require.NoError(t, db.Create(&mcmpapi.McmpApiAction{
		ServiceName: "mc-infra-manager", ActionName: "GetAllInfra", Method: "get", ResourcePath: "/ns/{nsId}/infra",
	}).Error)
	h := newMcmpApiCallTestHandler(db)
	req := &model.McmpApiCallRequest{
		ServiceName: "mc-infra-manager", ActionName: "GetAllInfra",
		RequestParams: model.McmpApiRequestParams{PathParams: map[string]string{"nsId": "ns01"}},
	}

	status, _ := h.authorizeMcmpApiAction(context.Background(), false, "", req)

	require.Equal(t, http.StatusUnauthorized, status, "kcUserID가 없으면 소유권 조회 전에 401로 거부해야 한다")
}

func TestAuthorizeNsOwnership_MemberOfOwningWorkspace_Passes(t *testing.T) {
	db := setupMcmpApiCallTestDB(t)
	workspace := &model.Workspace{Name: "tc-iam-tech-003-ws"}
	require.NoError(t, db.Create(workspace).Error)
	project := &model.Project{Name: "tc-iam-tech-003-proj", NsId: "ns-owned"}
	require.NoError(t, db.Create(project).Error)
	require.NoError(t, db.Exec(`INSERT INTO mcmp_workspace_projects (workspace_id, project_id) VALUES (?, ?)`,
		workspace.ID, project.ID).Error)
	require.NoError(t, db.Exec(`INSERT INTO mcmp_user_workspace_roles (user_id, workspace_id, role_id) VALUES (?, ?, ?)`,
		5, workspace.ID, 1).Error)
	h := newMcmpApiCallTestHandler(db)

	status, msg := h.authorizeNsOwnership("ns-owned", 5)

	require.Equal(t, 0, status, "프로젝트가 속한 워크스페이스의 멤버는 통과해야 한다: %s", msg)
}

func TestAuthorizeNsOwnership_NotMemberOfOwningWorkspace_ReturnsForbidden(t *testing.T) {
	db := setupMcmpApiCallTestDB(t)
	workspace := &model.Workspace{Name: "tc-iam-tech-003-ws-2"}
	require.NoError(t, db.Create(workspace).Error)
	project := &model.Project{Name: "tc-iam-tech-003-proj-2", NsId: "ns-not-owned"}
	require.NoError(t, db.Create(project).Error)
	require.NoError(t, db.Exec(`INSERT INTO mcmp_workspace_projects (workspace_id, project_id) VALUES (?, ?)`,
		workspace.ID, project.ID).Error)
	// user 5 belongs to a *different* workspace, not the one that owns this project
	otherWorkspace := &model.Workspace{Name: "tc-iam-tech-003-ws-other"}
	require.NoError(t, db.Create(otherWorkspace).Error)
	require.NoError(t, db.Exec(`INSERT INTO mcmp_user_workspace_roles (user_id, workspace_id, role_id) VALUES (?, ?, ?)`,
		5, otherWorkspace.ID, 1).Error)
	h := newMcmpApiCallTestHandler(db)

	status, _ := h.authorizeNsOwnership("ns-not-owned", 5)

	require.Equal(t, http.StatusForbidden, status, "프로젝트가 속한 워크스페이스에 소속되지 않은 사용자는 거부해야 한다")
}

func TestAuthorizeNsOwnership_UnknownNsId_ReturnsForbidden(t *testing.T) {
	h := newMcmpApiCallTestHandler(setupMcmpApiCallTestDB(t))

	status, _ := h.authorizeNsOwnership("ns-does-not-exist", 5)

	require.Equal(t, http.StatusForbidden, status, "존재하지 않는 nsId는 거부해야 한다")
}
