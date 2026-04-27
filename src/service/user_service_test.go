package service

// user_service_test.go
// 비활성화/탈퇴 관련 UserService 메서드 단위 테스트
//
// 테스트 범위:
//   - DeactivateUser: user-not-found, self-deactivate, already-inactive 검증
//   - ActivateUser:   user-not-found, not-inactive 검증
//   - RequestWithdrawal: user-not-found, non-active 검증 및 정상 상태 전이
//   - ProcessWithdrawal: user-not-found, wrong-status 검증 및 role-mapping 삭제 후 상태 전이
//
// NOTE: DeactivateUser / ActivateUser / ProcessWithdrawal 은 내부에서
//       NewKeycloakService()를 직접 호출하므로 Keycloak 이후 단계는
//       외부 서비스 없이 테스트할 수 없습니다.  해당 케이스는 통합 테스트로 분리합니다.

import (
	"context"
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── DB 헬퍼 ───────────────────────────────────────────────────────────────────

func setupUserServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// UserRepository.AutoMigrate 가 내부에서 User 만 마이그레이션하므로
	// DeleteAllRoleMappings 가 참조하는 테이블도 명시적으로 생성합니다.
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.RoleMaster{},
		&model.UserPlatformRole{},
		&model.UserWorkspaceRole{},
		&model.UserOrganization{},
	))
	return db
}

// newTestUserService 는 SQLite in-memory DB 를 사용하는 UserService 를 생성합니다.
func newTestUserService(t *testing.T) (*UserService, *gorm.DB) {
	t.Helper()
	db := setupUserServiceTestDB(t)

	userRepo := repository.NewUserRepository(db)

	svc := &UserService{
		db:       db,
		userRepo: userRepo,
	}
	return svc, db
}

// createTestUser 는 테스트용 사용자를 DB에 직접 삽입합니다.
func createTestUser(t *testing.T, db *gorm.DB, kcID string, status model.UserStatus) *model.User {
	t.Helper()
	u := &model.User{
		Username: "user_" + kcID,
		KcId:     kcID,
		Status:   status,
	}
	// GORM SQLite: Status zero-value 무시 방지를 위해 Map 사용
	require.NoError(t, db.Model(&model.User{}).Create(map[string]interface{}{
		"username": u.Username,
		"kc_id":    u.KcId,
		"status":   string(u.Status),
	}).Error)
	// 삽입된 레코드를 다시 읽어 ID 확보
	var created model.User
	require.NoError(t, db.Where("kc_id = ?", kcID).First(&created).Error)
	return &created
}

// ── DeactivateUser 테스트 ─────────────────────────────────────────────────────

// TC-DU-01: 존재하지 않는 사용자 ID → user not found 에러
func TestDeactivateUser_UserNotFound(t *testing.T) {
	svc, _ := newTestUserService(t)

	err := svc.DeactivateUser(context.Background(), 99999, "some-kc-id")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

// TC-DU-02: 요청자가 자기 자신을 비활성화하려는 경우 → cannot deactivate yourself
func TestDeactivateUser_SelfDeactivation(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-self-001", model.UserStatusActive)

	err := svc.DeactivateUser(context.Background(), user.ID, user.KcId)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot deactivate yourself")
}

// TC-DU-03: 이미 INACTIVE 상태인 사용자 → already inactive 에러
func TestDeactivateUser_AlreadyInactive(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-inactive-001", model.UserStatusInactive)

	err := svc.DeactivateUser(context.Background(), user.ID, "kc-requestor-001")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already inactive")
}

// ── ActivateUser 테스트 ───────────────────────────────────────────────────────

// TC-AU-01: 존재하지 않는 사용자 ID → user not found 에러
func TestActivateUser_UserNotFound(t *testing.T) {
	svc, _ := newTestUserService(t)

	err := svc.ActivateUser(context.Background(), 99999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

// TC-AU-02: ACTIVE 상태의 사용자를 활성화하려는 경우 → not inactive 에러
func TestActivateUser_UserNotInactive_Active(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-active-002", model.UserStatusActive)

	err := svc.ActivateUser(context.Background(), user.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not inactive")
}

// TC-AU-03: WITHDRAWAL_REQUESTED 상태 사용자 활성화 시도 → not inactive 에러
func TestActivateUser_UserNotInactive_WithdrawalRequested(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-wr-003", model.UserStatusWithdrawalRequested)

	err := svc.ActivateUser(context.Background(), user.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not inactive")
}

// ── RequestWithdrawal 테스트 ──────────────────────────────────────────────────

// TC-RW-01: 존재하지 않는 kcUserID → user not found 에러
func TestRequestWithdrawal_UserNotFound(t *testing.T) {
	svc, _ := newTestUserService(t)

	err := svc.RequestWithdrawal(context.Background(), "kc-nonexistent-999")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

// TC-RW-02: INACTIVE 상태 사용자 → only active users can request withdrawal 에러
func TestRequestWithdrawal_InactiveUser(t *testing.T) {
	svc, db := newTestUserService(t)
	createTestUser(t, db, "kc-inactive-rw-001", model.UserStatusInactive)

	err := svc.RequestWithdrawal(context.Background(), "kc-inactive-rw-001")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only active users can request withdrawal")
}

// TC-RW-03: WITHDRAWAL_REQUESTED 상태 사용자 재신청 → only active users can request withdrawal 에러
func TestRequestWithdrawal_AlreadyRequested(t *testing.T) {
	svc, db := newTestUserService(t)
	createTestUser(t, db, "kc-wr-rw-001", model.UserStatusWithdrawalRequested)

	err := svc.RequestWithdrawal(context.Background(), "kc-wr-rw-001")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only active users can request withdrawal")
}

// TC-RW-04: WITHDRAWN 상태 사용자 탈퇴 신청 → only active users can request withdrawal 에러
func TestRequestWithdrawal_AlreadyWithdrawn(t *testing.T) {
	svc, db := newTestUserService(t)
	createTestUser(t, db, "kc-withdrawn-rw-001", model.UserStatusWithdrawn)

	err := svc.RequestWithdrawal(context.Background(), "kc-withdrawn-rw-001")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only active users can request withdrawal")
}

// TC-RW-05: ACTIVE 사용자 정상 탈퇴 신청 → DB 상태 WITHDRAWAL_REQUESTED 로 변경
func TestRequestWithdrawal_Success(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-active-rw-ok", model.UserStatusActive)

	err := svc.RequestWithdrawal(context.Background(), user.KcId)

	require.NoError(t, err)

	// DB에서 직접 확인
	var updated model.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	assert.Equal(t, model.UserStatusWithdrawalRequested, updated.Status)
}

// ── ProcessWithdrawal 테스트 ──────────────────────────────────────────────────

// TC-PW-01: 존재하지 않는 사용자 ID → user not found 에러
func TestProcessWithdrawal_UserNotFound(t *testing.T) {
	svc, _ := newTestUserService(t)

	err := svc.ProcessWithdrawal(context.Background(), 99999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

// TC-PW-02: ACTIVE 상태 사용자 최종 처리 시도 → has not requested withdrawal 에러
func TestProcessWithdrawal_NotRequested_Active(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-active-pw-001", model.UserStatusActive)

	err := svc.ProcessWithdrawal(context.Background(), user.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "has not requested withdrawal")
}

// TC-PW-03: INACTIVE 상태 사용자 최종 처리 시도 → has not requested withdrawal 에러
func TestProcessWithdrawal_NotRequested_Inactive(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-inactive-pw-001", model.UserStatusInactive)

	err := svc.ProcessWithdrawal(context.Background(), user.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "has not requested withdrawal")
}

// TC-PW-04: WITHDRAWAL_REQUESTED 상태 → role 매핑 삭제 후 Keycloak 호출 전까지 성공
//
// NOTE: NewKeycloakService() 를 직접 호출하므로 Keycloak 연결 없이는
//       ProcessWithdrawal 전체 성공 경로를 단위 테스트로 검증할 수 없습니다.
//       이 테스트는 DB 레이어(role 매핑 삭제)까지만 검증하고,
//       Keycloak 호출 실패 에러 메시지로 경계를 확인합니다.
func TestProcessWithdrawal_RoleMappingsDeletedBeforeKC(t *testing.T) {
	svc, db := newTestUserService(t)
	user := createTestUser(t, db, "kc-wr-pw-ok", model.UserStatusWithdrawalRequested)

	// 플랫폼 롤 매핑 삽입
	require.NoError(t, db.Create(&model.UserPlatformRole{UserID: user.ID, RoleID: 1}).Error)

	err := svc.ProcessWithdrawal(context.Background(), user.ID)

	// Keycloak 미연결이므로 "failed to disable user in keycloak" 에러가 나야 정상
	// (그 이전 role 매핑 삭제는 성공했음을 간접 확인)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to disable user in keycloak",
		"role 매핑 삭제 이후 Keycloak 단계에서 실패해야 합니다")

	// role 매핑이 실제로 삭제됐는지 확인
	var count int64
	db.Model(&model.UserPlatformRole{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(0), count, "platform role 매핑이 삭제되어야 합니다")
}
