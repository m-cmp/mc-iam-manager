package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// recordingKeycloakService는 DeleteRealmRole 호출 여부/인자만 기록하는 테스트 스텁이다.
// service.KeycloakService를 nil로 임베드해 나머지 메서드는 오버라이드하지 않고도
// 인터페이스를 만족시킨다 — 이 테스트가 실제로 부르는 건 DeleteRealmRole 뿐이다.
type recordingKeycloakService struct {
	service.KeycloakService
	deleteRealmRoleCalls []string
}

func (m *recordingKeycloakService) DeleteRealmRole(ctx context.Context, roleName string) error {
	m.deleteRealmRoleCalls = append(m.deleteRealmRoleCalls, roleName)
	return nil
}

func setupDeleteRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Named per-test in-memory DB (not the bare `file::memory:?cache=shared` used
	// elsewhere) so this test's schema doesn't collide with model.User's
	// many2many tags, which target these same join table names
	// (mcmp_user_platform_roles/mcmp_user_workspace_roles) with a narrower,
	// implicitly-created schema when some other test in this package migrates
	// model.User against the shared cache.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.RoleMasterCspRoleMapping{},
		&model.RoleMenuMapping{},
	))
	// model.UserPlatformRole/UserWorkspaceRole embed a `User`/`Role` belongsTo
	// association field, which forces gorm to parse model.User — whose
	// many2many tag targets these exact table names and collides with
	// AutoMigrate, silently truncating them to a 2-column join table. Create
	// the real columns by hand instead of via AutoMigrate for these two.
	require.NoError(t, db.Exec(`CREATE TABLE mcmp_user_platform_roles (
		user_id integer, role_id integer, created_at datetime, username text,
		PRIMARY KEY (user_id, role_id)
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE mcmp_user_workspace_roles (
		user_id integer, workspace_id integer, role_id integer,
		username text, workspace_name text, role_name text, created_at datetime,
		PRIMARY KEY (user_id, workspace_id, role_id)
	)`).Error)
	return db
}

func newDeleteRoleContext(t *testing.T, roleID uint, roleType string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	body := fmt.Sprintf(`{"roleType":%q}`, roleType)
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/id/x", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(fmt.Sprint(roleID))
	return c, rec
}

// TC-IAM-TECH-025-01: platform 타입 서브를 가진 역할을 삭제하면 Keycloak realm role도
// 함께(best-effort) 삭제 시도되어야 한다.
func TestDeleteRole_PlatformRole_DeletesKeycloakRealmRole(t *testing.T) {
	db := setupDeleteRoleTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-tech-025-platform"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)

	h := NewRoleHandler(db)
	kc := &recordingKeycloakService{}
	h.keycloakService = kc

	c, rec := newDeleteRoleContext(t, role.ID, "platform")
	require.NoError(t, h.DeleteRole(c))
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, []string{"tc-iam-tech-025-platform"}, kc.deleteRealmRoleCalls,
		"platform 역할 삭제 시 동일 이름의 Keycloak realm role 삭제가 시도되어야 한다")

	var count int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&count).Error)
	require.Zero(t, count, "역할 자체는 정상적으로 삭제되어야 한다")
}

// TC-IAM-TECH-025-02: workspace 타입 서브만 가진 역할은 realm role이 애초에 생성되지
// 않는 경로이므로, 삭제 시 Keycloak 호출을 시도하지 않아야 한다.
func TestDeleteRole_WorkspaceOnlyRole_DoesNotDeleteKeycloakRealmRole(t *testing.T) {
	db := setupDeleteRoleTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-tech-025-workspace"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeWorkspace}).Error)

	h := NewRoleHandler(db)
	kc := &recordingKeycloakService{}
	h.keycloakService = kc

	c, rec := newDeleteRoleContext(t, role.ID, "workspace")
	require.NoError(t, h.DeleteRole(c))
	require.Equal(t, http.StatusOK, rec.Code)

	require.Empty(t, kc.deleteRealmRoleCalls,
		"platform sub가 없는 역할은 Keycloak realm role 삭제를 시도하면 안 된다")
}

// TC-IAM-TECH-025-03: 하나의 role_id가 platform+workspace 서브를 동시에 가질 때,
// workspace roleType으로 삭제 요청이 들어와도(응답 role.RoleSubs는 workspace로만
// 필터되어 preload됨) platform sub 존재 여부는 별도로 확인해 realm role을 정리해야 한다.
func TestDeleteRole_MixedRoleTypes_RequestScopedToWorkspace_StillCleansUpRealmRole(t *testing.T) {
	db := setupDeleteRoleTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-tech-025-mixed"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeWorkspace}).Error)

	h := NewRoleHandler(db)
	kc := &recordingKeycloakService{}
	h.keycloakService = kc

	// 삭제 요청 자체는 workspace roleType으로 스코프됨
	c, rec := newDeleteRoleContext(t, role.ID, "workspace")
	require.NoError(t, h.DeleteRole(c))
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, []string{"tc-iam-tech-025-mixed"}, kc.deleteRealmRoleCalls,
		"요청이 workspace로 스코프돼도 같은 role_id에 platform sub가 있었다면 realm role을 정리해야 한다")
}
