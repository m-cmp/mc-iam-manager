package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
)

// TC-IAM-TECH-035-01: Create가 icon 값을 그대로 저장해야 한다.
func TestCreate_IconRoundTrip(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	s := NewMenuService(db)

	req := &model.CreateMenuRequest{
		ID:          "tc-iam-tech-035-create",
		DisplayName: "Icon Test",
		ResType:     "menu",
		Priority:    "1",
		MenuNumber:  "1",
		Icon:        "mdi-home",
	}
	require.NoError(t, s.Create(req))

	var menu model.Menu
	require.NoError(t, db.Where("id = ?", req.ID).First(&menu).Error)
	require.Equal(t, "mdi-home", menu.Icon)
}

// TC-IAM-TECH-035-02: CreateWithRoleMappings가 icon 값을 그대로 저장해야 한다.
func TestCreateWithRoleMappings_IconRoundTrip(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	seedAdminPlatformRole(t, db)
	s := NewMenuService(db)

	req := &model.CreateMenuRequest{
		ID:          "tc-iam-tech-035-create-with-role",
		DisplayName: "Icon Test With Role",
		ResType:     "menu",
		Priority:    "1",
		MenuNumber:  "1",
		Icon:        "mdi-cog",
	}
	resp, err := s.CreateWithRoleMappings(req)
	require.NoError(t, err)
	require.Equal(t, "mdi-cog", resp.Menu.Icon)

	var menu model.Menu
	require.NoError(t, db.Where("id = ?", req.ID).First(&menu).Error)
	require.Equal(t, "mdi-cog", menu.Icon)
}

// TC-IAM-TECH-035-03: Update가 icon만 부분 갱신할 수 있어야 한다(다른 필드는 유지).
func TestUpdate_IconRoundTrip(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	s := NewMenuService(db)

	req := &model.CreateMenuRequest{
		ID:          "tc-iam-tech-035-update",
		DisplayName: "Before Update",
		ResType:     "menu",
		Priority:    "1",
		MenuNumber:  "1",
		Icon:        "mdi-old",
	}
	require.NoError(t, s.Create(req))

	require.NoError(t, s.Update(req.ID, map[string]interface{}{"icon": "mdi-new"}))

	var menu model.Menu
	require.NoError(t, db.Where("id = ?", req.ID).First(&menu).Error)
	require.Equal(t, "mdi-new", menu.Icon)
	require.Equal(t, "Before Update", menu.DisplayName, "unrelated fields must be untouched by a partial update")
}
