package repository

// role_repository_test.go
//
// FindCspRoleByName 단위 테스트 (SQLite in-memory DB) — "CSP 역할 없음" 오분류 회귀.
// .First()가 못 찾으면 gorm.ErrRecordNotFound를 그대로 흘려보내던 것을, 바로 옆 함수
// GetCspRolesByName이 이미 쓰던 패턴(ErrRecordNotFound -> nil, nil)으로 맞췄다.
// 핸들러(GetCspRoleByName)는 err != nil -> 400, role == nil -> 404 순으로 분기하므로,
// 이 수정 전에는 "없음"이 항상 400(+오타 메시지)으로만 나가고 404 분기는 죽은 코드였다.

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRoleRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CspRole{}))
	return db
}

// TC-FCRBN-01: 존재하지 않는 이름 -> (nil, nil) — err가 나면 안 된다(핸들러가 400으로 오분류하던 원인)
func TestFindCspRoleByName_NotFound(t *testing.T) {
	db := setupRoleRepoTestDB(t)
	repo := NewRoleRepository(db)

	role, err := repo.FindCspRoleByName("mciam-does-not-exist")

	require.NoError(t, err, "없음은 에러가 아니라 (nil, nil)이어야 핸들러의 404 분기가 살아난다")
	assert.Nil(t, role)
}

// TC-FCRBN-02: 존재하는 이름 -> 해당 role 반환
func TestFindCspRoleByName_Found(t *testing.T) {
	db := setupRoleRepoTestDB(t)
	repo := NewRoleRepository(db)
	require.NoError(t, db.Create(&model.CspRole{Name: "mciam-fcrbn02", CspType: "aws"}).Error)

	role, err := repo.FindCspRoleByName("mciam-fcrbn02")

	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Equal(t, "mciam-fcrbn02", role.Name)
}
