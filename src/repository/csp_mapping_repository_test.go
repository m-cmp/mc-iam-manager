package repository

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCspMappingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.CspAccount{},
		&model.CspIdpConfig{},
		&model.CspRole{},
		&model.RoleMasterCspRoleMapping{},
	))
	return db
}

func seedCspRoleMapping(t *testing.T, db *gorm.DB, roleID uint, authMethod constants.AuthMethod, cspRoleName string) *model.CspRole {
	t.Helper()
	cspRole := &model.CspRole{Name: cspRoleName, CspType: "aws", IamIdentifier: "arn:aws:iam::111111111111:role/" + cspRoleName}
	require.NoError(t, db.Create(cspRole).Error)
	require.NoError(t, db.Create(&model.RoleMasterCspRoleMapping{RoleID: roleID, AuthMethod: authMethod, CspRoleID: cspRole.ID}).Error)
	return cspRole
}

func TestFindCspRoleMappingsByRoleIDAndCspType_Single(t *testing.T) {
	db := setupCspMappingTestDB(t)
	r := NewCspMappingRepository(db)
	cspRole := seedCspRoleMapping(t, db, 1, constants.AuthMethodOIDC, "mciam-admin")

	m, err := r.FindCspRoleMappingsByRoleIDAndCspType(1, "aws", "OIDC")
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Len(t, m.CspRoles, 1)
	assert.Equal(t, cspRole.ID, m.CspRoles[0].ID)

	m, err = r.FindCspRoleMappingsByRoleIDAndCspType(2, "aws", "OIDC")
	require.NoError(t, err)
	assert.Nil(t, m)
}

// 매핑 대상 CspRole 이 2개 이상이면 임의로 고르지 않는다.
func TestFindCspRoleMappingsByRoleIDAndCspType_AmbiguousCspRoles(t *testing.T) {
	db := setupCspMappingTestDB(t)
	r := NewCspMappingRepository(db)
	seedCspRoleMapping(t, db, 1, constants.AuthMethodOIDC, "mciam-admin-a")
	seedCspRoleMapping(t, db, 1, constants.AuthMethodOIDC, "mciam-admin-b")

	m, err := r.FindCspRoleMappingsByRoleIDAndCspType(1, "aws", "OIDC")
	assert.ErrorIs(t, err, ErrAmbiguousCspRoleMapping)
	assert.Nil(t, m)
}

// 같은 CspRole 을 OIDC·SAML 둘 다로 매핑한 경우 authMethod 미지정이어도 대상이 하나라 모호하지 않다.
func TestFindCspRoleMappingsByRoleIDAndCspType_SameCspRoleTwoAuthMethods(t *testing.T) {
	db := setupCspMappingTestDB(t)
	r := NewCspMappingRepository(db)
	cspRole := seedCspRoleMapping(t, db, 1, constants.AuthMethodOIDC, "mciam-admin")
	require.NoError(t, db.Create(&model.RoleMasterCspRoleMapping{RoleID: 1, AuthMethod: constants.AuthMethodSAML, CspRoleID: cspRole.ID}).Error)

	m, err := r.FindCspRoleMappingsByRoleIDAndCspType(1, "aws", "")
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, cspRole.ID, m.CspRoles[0].ID)

	// 인증 방식을 지정하면 그 방식의 매핑만 본다
	other := seedCspRoleMapping(t, db, 1, constants.AuthMethodSAML, "mciam-admin-saml")
	_, err = r.FindCspRoleMappingsByRoleIDAndCspType(1, "aws", "SAML")
	assert.ErrorIs(t, err, ErrAmbiguousCspRoleMapping)
	m, err = r.FindCspRoleMappingsByRoleIDAndCspType(1, "aws", "OIDC")
	require.NoError(t, err)
	assert.Equal(t, cspRole.ID, m.CspRoles[0].ID)
	assert.NotEqual(t, other.ID, m.CspRoles[0].ID)
}
