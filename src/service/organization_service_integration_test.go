package service

// organization_service_integration_test.go
//
// 조직 계층 구조(Organization Hierarchy) OrganizationService 메서드 중
// PostgreSQL 전용 SQL(CTE, ::text 캐스팅 등)을 사용하는 FindUserOrganizations
// 경로를 실제 PostgreSQL 에 대해 검증하는 통합 테스트.
//
// organization_service_test.go 의 SQLite 기반 단위 테스트에서는 실행할 수 없어
// 이 파일로 이동되었습니다 (GetUserOrganizations, ReplaceUserGroups 의 정상 경로).
//
// 실행 방법 (csp_credential_integration_test.go 와 동일한 패턴/게이팅):
//   INTEGRATION_TEST=1 \
//   MC_IAM_MANAGER_DATABASE_HOST=... MC_IAM_MANAGER_DATABASE_PORT=... \
//   MC_IAM_MANAGER_DATABASE_USER=... MC_IAM_MANAGER_DATABASE_PASSWORD=... \
//   MC_IAM_MANAGER_DATABASE_NAME=... MC_IAM_MANAGER_DATABASE_SSLMODE=... \
//   go test github.com/m-cmp/mc-iam-manager/service -run "TestOrgServicePG" -v -count=1
//
// DB 접속 정보는 config.NewDatabaseConfig() (프로덕션과 동일한 코드 경로)로 읽습니다.
// DB에 연결할 수 없으면 skip 됩니다 (SQLite로 폴백하지 않습니다).
//
// 각 테스트는 트랜잭션을 시작하고 종료 시 롤백하여, 실제(영속) PostgreSQL DB를
// 사용하면서도 테스트 간 독립성과 재실행 가능성을 보장합니다.

import (
	"sync"
	"testing"

	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var orgServicePGMigrateOnce sync.Once

// setupOrgServicePGTestDB 실제 PostgreSQL에 연결하고, 테스트에 필요한 테이블을
// (최초 1회) AutoMigrate 한 뒤, 트랜잭션을 열어 반환합니다.
// 트랜잭션은 테스트 종료 시 자동 롤백되어 DB 상태를 원복합니다.
func setupOrgServicePGTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	skipIfNotIntegration(t)

	dbConfig := config.NewDatabaseConfig()
	db, err := gorm.Open(postgres.Open(dbConfig.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("DB 연결 실패 — DB 미연결로 테스트를 건너뜁니다: %v", err)
	}

	orgServicePGMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&model.Organization{},
			&model.User{},
			&model.UserOrganization{},
		))
	})

	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() {
		tx.Rollback()
	})
	return tx
}

func newOrgServicePGTest(t *testing.T) (*OrganizationService, *gorm.DB) {
	t.Helper()
	db := setupOrgServicePGTestDB(t)
	svc := NewOrganizationService(db)
	return svc, db
}

// ── GetUserOrganizations 테스트 (PostgreSQL) ─────────────────────────────────

// TC-GU-01: 소속 조직 없는 사용자 → 빈 슬라이스
func TestOrgServicePG_GetUserOrganizations_Empty(t *testing.T) {
	svc, _ := newOrgServicePGTest(t)

	orgs, err := svc.GetUserOrganizations(99999)

	require.NoError(t, err)
	assert.Empty(t, orgs)
}

// TC-GU-02: 소속 조직 목록 정상 반환
func TestOrgServicePG_GetUserOrganizations_WithOrgs(t *testing.T) {
	svc, db := newOrgServicePGTest(t)
	org1 := createOrg(t, db, "01", "Alpha", nil)
	org2 := createOrg(t, db, "02", "Beta", nil)
	user := createOrgUser(t, db, "frank", "kc-frank-01")

	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org1.ID}).Error)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org2.ID}).Error)

	orgs, err := svc.GetUserOrganizations(user.ID)

	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

// ── ReplaceUserGroups 테스트 (PostgreSQL) ────────────────────────────────────

// TC-RG-02: 기존 그룹 전부 제거 후 신규 할당 (빈 배열)
func TestOrgServicePG_ReplaceUserGroups_RemoveAll(t *testing.T) {
	svc, db := newOrgServicePGTest(t)
	org := createOrg(t, db, "01", "Old", nil)
	user := createOrgUser(t, db, "grace", "kc-grace-01")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org.ID}).Error)

	err := svc.ReplaceUserGroups(user.ID, []uint{})

	require.NoError(t, err)

	var count int64
	db.Model(&model.UserOrganization{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TC-RG-03: 기존 그룹 교체 성공
func TestOrgServicePG_ReplaceUserGroups_Replace(t *testing.T) {
	svc, db := newOrgServicePGTest(t)
	org1 := createOrg(t, db, "01", "OldGroup", nil)
	org2 := createOrg(t, db, "02", "NewGroup", nil)
	user := createOrgUser(t, db, "henry", "kc-henry-01")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org1.ID}).Error)

	err := svc.ReplaceUserGroups(user.ID, []uint{org2.ID})

	require.NoError(t, err)

	orgs, err := svc.GetUserOrganizations(user.ID)
	require.NoError(t, err)
	require.Len(t, orgs, 1)
	assert.Equal(t, "NewGroup", orgs[0].Name)
}
