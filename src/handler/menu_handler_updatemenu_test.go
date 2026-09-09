package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMenuHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Menu{}))
	return db
}

// TC-IAM-TECH-035-04: MenuHandler.UpdateMenu가 icon 필드를 응답에서 왕복시켜야 한다
// (UpsertMenus의 DoUpdates 컬럼 리스트 누락과는 다른 경로지만, 핸들러가 요청의 icon을
// updates 맵에 실어 보내는 라인 자체를 검증).
func TestMenuHandlerUpdateMenu_IconRoundTrip(t *testing.T) {
	db := setupMenuHandlerTestDB(t)
	require.NoError(t, db.Create(&model.Menu{
		ID:          "tc-iam-tech-035-handler-update",
		DisplayName: "Before",
		ResType:     "menu",
		Priority:    1,
		MenuNumber:  1,
		Icon:        "mdi-old",
	}).Error)

	h := NewMenuHandler(db)

	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}

	body := `{"icon":"mdi-new-handler"}`
	req := httptest.NewRequest(http.MethodPut, "/api/menus/id/tc-iam-tech-035-handler-update", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("menuId")
	c.SetParamValues("tc-iam-tech-035-handler-update")

	require.NoError(t, h.UpdateMenu(c))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"icon":"mdi-new-handler"`)

	var menu model.Menu
	require.NoError(t, db.Where("id = ?", "tc-iam-tech-035-handler-update").First(&menu).Error)
	require.Equal(t, "mdi-new-handler", menu.Icon)
}
