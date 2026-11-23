package repository

import (
	"testing"
	"time"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 역할 수정은 이름·설명·상위 역할만 바꾸고 created_at·predefined 는 보존한다
func TestUpdateRoleWithSubsWithTx_PreservesUntouchedColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.RoleMaster{}, &model.RoleSub{}))
	created := time.Date(2026, 10, 26, 1, 29, 0, 0, time.UTC)
	role := &model.RoleMaster{Name: "t-upd", Description: "old", Predefined: true, CreatedAt: created}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)

	r := NewRoleRepository(db)
	updated, err := r.UpdateRoleWithSubsWithTx(db, model.RoleMaster{ID: role.ID, Name: "t-upd2", Description: "new"},
		[]constants.IAMRoleType{constants.RoleTypePlatform, constants.RoleTypeCSP})
	require.NoError(t, err)
	assert.Equal(t, "t-upd2", updated.Name)
	assert.Equal(t, "new", updated.Description)
	assert.True(t, updated.Predefined, "predefined 보존")
	assert.True(t, updated.CreatedAt.Equal(created), "created_at 보존: %v", updated.CreatedAt)
	assert.Len(t, updated.RoleSubs, 2)
}
