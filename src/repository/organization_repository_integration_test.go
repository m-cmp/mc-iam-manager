package repository

// organization_repository_integration_test.go
//
// OrganizationRepository 중 PostgreSQL 전용 SQL(CTE, ::text 캐스팅,
// SUBSTRING ... FROM, ILIKE)을 사용하는 메서드들에 대한 실제 PostgreSQL
// 통합 테스트입니다. 아래 메서드들은 SQLite에서 실행할 수 없어 그동안
// 테스트 커버리지가 전혀 없었습니다:
//   - FindTreeFlat
//   - FindSubtreeFlat
//   - GetSubtreeDepth
//   - GetAncestorDepth
//   - UpdateDescendantCodes
//   - FindByFilter (ILIKE)
//
// 실행 방법 (service 패키지의 csp_credential_integration_test.go 와 동일한 패턴):
//   INTEGRATION_TEST=1 \
//   MC_IAM_MANAGER_DATABASE_HOST=... MC_IAM_MANAGER_DATABASE_PORT=... \
//   MC_IAM_MANAGER_DATABASE_USER=... MC_IAM_MANAGER_DATABASE_PASSWORD=... \
//   MC_IAM_MANAGER_DATABASE_NAME=... MC_IAM_MANAGER_DATABASE_SSLMODE=... \
//   go test github.com/m-cmp/mc-iam-manager/repository -run "TestOrgRepoPG" -v -count=1
//
// DB 접속 정보는 config.NewDatabaseConfig() (프로덕션과 동일한 코드 경로)로 읽으며,
// DB에 연결할 수 없으면 skip 됩니다 (SQLite로 폴백하지 않습니다).
//
// 각 테스트는 트랜잭션을 시작하고 종료 시 롤백하여, 실제(영속) PostgreSQL DB를
// 사용하면서도 테스트 간 독립성과 재실행 가능성을 보장합니다.

import (
	"os"
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

var orgRepoPGMigrateOnce sync.Once

// skipIfNotIntegrationOrgRepo 통합 테스트 환경이 아니면 skip
// (service 패키지의 skipIfNotIntegration 과 동일한 게이팅 규칙)
func skipIfNotIntegrationOrgRepo(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("INTEGRATION_TEST=1 이 설정되지 않아 통합 테스트를 건너뜁니다")
	}
}

// setupOrgRepoPGTestDB 실제 PostgreSQL에 연결하고, 테스트에 필요한 테이블을
// (최초 1회) AutoMigrate 한 뒤, 트랜잭션을 열어 반환합니다.
// 트랜잭션은 테스트 종료 시 자동 롤백되어 DB 상태를 원복합니다.
func setupOrgRepoPGTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	skipIfNotIntegrationOrgRepo(t)

	dbConfig := config.NewDatabaseConfig()
	db, err := gorm.Open(postgres.Open(dbConfig.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("DB 연결 실패 — DB 미연결로 테스트를 건너뜁니다: %v", err)
	}

	orgRepoPGMigrateOnce.Do(func() {
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

func newOrgRepoPGTest(t *testing.T) (*OrganizationRepository, *gorm.DB) {
	t.Helper()
	db := setupOrgRepoPGTestDB(t)
	return NewOrganizationRepository(db), db
}

func createTestOrg(t *testing.T, db *gorm.DB, code, name string, parentID *uint) *model.Organization {
	t.Helper()
	org := &model.Organization{
		OrganizationCode: code,
		Name:             name,
		ParentID:         parentID,
	}
	require.NoError(t, db.Create(org).Error)
	return org
}

// ── FindTreeFlat ──────────────────────────────────────────────────────────────

// TC-FTF-01: 조직이 하나도 없으면 빈 슬라이스 반환
func TestOrgRepoPG_FindTreeFlat_Empty(t *testing.T) {
	repo, _ := newOrgRepoPGTest(t)

	trees, err := repo.FindTreeFlat()

	require.NoError(t, err)
	assert.Empty(t, trees)
}

// TC-FTF-02: 3단계 계층 구조 → level/path 정확히 계산
func TestOrgRepoPG_FindTreeFlat_MultiLevel(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	root := createTestOrg(t, db, "01", "Root", nil)
	child := createTestOrg(t, db, "0101", "Child", &root.ID)
	grandchild := createTestOrg(t, db, "010101", "GrandChild", &child.ID)

	trees, err := repo.FindTreeFlat()
	require.NoError(t, err)
	require.Len(t, trees, 3)

	byID := make(map[uint]model.OrganizationTree, len(trees))
	for _, node := range trees {
		byID[node.ID] = node
	}

	require.Contains(t, byID, root.ID)
	require.Contains(t, byID, child.ID)
	require.Contains(t, byID, grandchild.ID)

	assert.Equal(t, 1, byID[root.ID].Level)
	assert.Equal(t, "/Root", byID[root.ID].Path)

	assert.Equal(t, 2, byID[child.ID].Level)
	assert.Equal(t, "/Root/Child", byID[child.ID].Path)

	assert.Equal(t, 3, byID[grandchild.ID].Level)
	assert.Equal(t, "/Root/Child/GrandChild", byID[grandchild.ID].Path)
}

// ── FindSubtreeFlat ───────────────────────────────────────────────────────────

// TC-FSF-01: 단일(리프) 조직 → 자신만 포함
func TestOrgRepoPG_FindSubtreeFlat_SingleLevel(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	org := createTestOrg(t, db, "01", "Leaf", nil)

	trees, err := repo.FindSubtreeFlat(org.ID)
	require.NoError(t, err)
	require.Len(t, trees, 1)
	assert.Equal(t, org.ID, trees[0].ID)
	assert.Equal(t, 1, trees[0].Level)
}

// TC-FSF-02: 다단계 계층에서 중간 노드를 루트로 조회 → 자신 + 하위만 포함 (형제/상위 제외)
func TestOrgRepoPG_FindSubtreeFlat_MultiLevel(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	root := createTestOrg(t, db, "01", "Root", nil)
	child := createTestOrg(t, db, "0101", "Child", &root.ID)
	grandchild := createTestOrg(t, db, "010101", "GrandChild", &child.ID)
	// 형제 조직 (subtree에 포함되면 안 됨)
	createTestOrg(t, db, "0102", "Sibling", &root.ID)

	trees, err := repo.FindSubtreeFlat(child.ID)
	require.NoError(t, err)
	require.Len(t, trees, 2)

	ids := []uint{trees[0].ID, trees[1].ID}
	assert.Contains(t, ids, child.ID)
	assert.Contains(t, ids, grandchild.ID)
	assert.NotContains(t, ids, root.ID)
}

// ── GetSubtreeDepth ───────────────────────────────────────────────────────────

// TC-GSD-01: 리프 조직(하위 없음) → depth 1
func TestOrgRepoPG_GetSubtreeDepth_Leaf(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	org := createTestOrg(t, db, "01", "Leaf", nil)

	depth, err := repo.GetSubtreeDepth(org.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, depth)
}

// TC-GSD-02: 3단계 하위 트리 → depth 3
func TestOrgRepoPG_GetSubtreeDepth_MultiLevel(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	root := createTestOrg(t, db, "01", "Root", nil)
	child := createTestOrg(t, db, "0101", "Child", &root.ID)
	createTestOrg(t, db, "010101", "GrandChild", &child.ID)

	depth, err := repo.GetSubtreeDepth(root.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, depth)
}

// ── GetAncestorDepth ──────────────────────────────────────────────────────────

// TC-GAD-01: 최상위 조직 → depth 1
func TestOrgRepoPG_GetAncestorDepth_Root(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	org := createTestOrg(t, db, "01", "Root", nil)

	depth, err := repo.GetAncestorDepth(org.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, depth)
}

// TC-GAD-02: 3단계 깊이의 자손 조직 → depth 3
func TestOrgRepoPG_GetAncestorDepth_MultiLevel(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	root := createTestOrg(t, db, "01", "Root", nil)
	child := createTestOrg(t, db, "0101", "Child", &root.ID)
	grandchild := createTestOrg(t, db, "010101", "GrandChild", &child.ID)

	depth, err := repo.GetAncestorDepth(grandchild.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, depth)
}

// ── UpdateDescendantCodes ─────────────────────────────────────────────────────

// TC-UDC-01: prefix 변경 시 하위 조직들의 코드만 변경, 자신의 코드는 변경되지 않음
func TestOrgRepoPG_UpdateDescendantCodes_PrefixRename(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	root := createTestOrg(t, db, "01", "Root", nil)
	child := createTestOrg(t, db, "0101", "Child", &root.ID)
	grandchild := createTestOrg(t, db, "010102", "GrandChild", &child.ID)

	err := repo.UpdateDescendantCodes("01", "09")
	require.NoError(t, err)

	var reloadedRoot, reloadedChild, reloadedGrandchild model.Organization
	require.NoError(t, db.First(&reloadedRoot, root.ID).Error)
	require.NoError(t, db.First(&reloadedChild, child.ID).Error)
	require.NoError(t, db.First(&reloadedGrandchild, grandchild.ID).Error)

	// UpdateDescendantCodes는 자신(oldPrefix와 완전히 일치하는 코드)은 건드리지 않음
	assert.Equal(t, "01", reloadedRoot.OrganizationCode)
	assert.Equal(t, "0901", reloadedChild.OrganizationCode)
	assert.Equal(t, "090102", reloadedGrandchild.OrganizationCode)
}

// TC-UDC-02: 대상 prefix를 가진 하위 조직이 없으면 아무 것도 변경되지 않고 오류도 없음
func TestOrgRepoPG_UpdateDescendantCodes_NoMatch(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	org := createTestOrg(t, db, "01", "Solo", nil)

	err := repo.UpdateDescendantCodes("99", "88")
	require.NoError(t, err)

	var reloaded model.Organization
	require.NoError(t, db.First(&reloaded, org.ID).Error)
	assert.Equal(t, "01", reloaded.OrganizationCode)
}

// ── FindByFilter ──────────────────────────────────────────────────────────────

// TC-FBF-01: name 대소문자 무관 부분일치 검색 (ILIKE)
func TestOrgRepoPG_FindByFilter_NameCaseInsensitivePartial(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	createTestOrg(t, db, "01", "Engineering", nil)
	createTestOrg(t, db, "02", "Marketing", nil)

	results, err := repo.FindByFilter("engine", "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Engineering", results[0].Name)
}

// TC-FBF-02: code 부분일치 검색
func TestOrgRepoPG_FindByFilter_CodePartial(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	createTestOrg(t, db, "0101", "Backend", nil)
	createTestOrg(t, db, "0202", "Frontend", nil)

	results, err := repo.FindByFilter("", "01")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Backend", results[0].Name)
}

// TC-FBF-03: 조건에 맞는 조직이 없으면 빈 슬라이스 반환
func TestOrgRepoPG_FindByFilter_NoMatch(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	createTestOrg(t, db, "01", "Sales", nil)

	results, err := repo.FindByFilter("nonexistent", "")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TC-FBF-04: name/code 모두 빈 값이면 전체 목록 반환
func TestOrgRepoPG_FindByFilter_EmptyFilterReturnsAll(t *testing.T) {
	repo, db := newOrgRepoPGTest(t)

	createTestOrg(t, db, "01", "Alpha", nil)
	createTestOrg(t, db, "02", "Beta", nil)

	results, err := repo.FindByFilter("", "")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
