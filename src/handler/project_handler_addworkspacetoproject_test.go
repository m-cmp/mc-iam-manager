package handler

import (
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

func setupProjectWorkspaceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Workspace{}, &model.Project{}))
	return db
}

// IAM-BUG-034: AddWorkspaceToProject는 DB write 성공 후 조건 없는 500 return이 남아 있어
// 항상 500을 반환했다. 성공 시 204 No Content를 반환해야 한다.
func TestAddWorkspaceToProject_Success_ReturnsNoContent(t *testing.T) {
	db := setupProjectWorkspaceTestDB(t)
	h := NewProjectHandler(db)

	workspace := &model.Workspace{Name: "tc-iam-bug-034-workspace"}
	require.NoError(t, db.Create(workspace).Error)
	project := &model.Project{Name: "tc-iam-bug-034-project"}
	require.NoError(t, db.Create(project).Error)

	e := echo.New()
	body := `{"workspaceId":"` + uintToStr(workspace.ID) + `","projectIds":["` + uintToStr(project.ID) + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/assign/workspaces", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.AddWorkspaceToProject(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

	var count int64
	require.NoError(t, db.Table("mcmp_workspace_projects").
		Where("workspace_id = ? AND project_id = ?", workspace.ID, project.ID).
		Count(&count).Error)
	require.Equal(t, int64(1), count, "워크스페이스-프로젝트 매핑이 생성되어야 한다")
}
