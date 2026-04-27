package repository

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCompanyTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.Company{})
	require.NoError(t, err)

	return db
}

func TestCompanyRepository_Create(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	company := &model.Company{
		Name:           "Test Company",
		RealmName:      "test-realm",
		KcClientID:     "test-client-id",
		KcClientSecret: "test-secret",
		Status:         "active",
	}

	err := repo.Create(company)
	assert.NoError(t, err)
	assert.NotZero(t, company.ID)
}

func TestCompanyRepository_First_NotFound(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	company, err := repo.First()
	assert.NoError(t, err)
	assert.Nil(t, company)
}

func TestCompanyRepository_First_Found(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	expected := &model.Company{
		Name:      "Test Company",
		RealmName: "test-realm",
		Status:    "active",
	}
	require.NoError(t, repo.Create(expected))

	company, err := repo.First()
	assert.NoError(t, err)
	require.NotNil(t, company)
	assert.Equal(t, expected.Name, company.Name)
	assert.Equal(t, expected.RealmName, company.RealmName)
	assert.Equal(t, "active", company.Status)
}

func TestCompanyRepository_ExistsByRealmName_False(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	exists, err := repo.ExistsByRealmName("non-existent-realm")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCompanyRepository_ExistsByRealmName_True(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	company := &model.Company{Name: "Company", RealmName: "my-realm", Status: "active"}
	require.NoError(t, repo.Create(company))

	exists, err := repo.ExistsByRealmName("my-realm")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCompanyRepository_Save(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	company := &model.Company{Name: "Original", RealmName: "realm", Status: "active"}
	require.NoError(t, repo.Create(company))

	company.Name = "Updated"
	company.Status = "inactive"
	err := repo.Save(company)
	assert.NoError(t, err)

	updated, err := repo.First()
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "inactive", updated.Status)
}

func TestCompanyRepository_Count_Empty(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	count, err := repo.Count()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestCompanyRepository_Count_WithData(t *testing.T) {
	db := setupCompanyTestDB(t)
	repo := NewCompanyRepository(db)

	require.NoError(t, repo.Create(&model.Company{Name: "C", RealmName: "realm1", Status: "active"}))

	count, err := repo.Count()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
