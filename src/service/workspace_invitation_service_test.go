package service

// workspace_invitation_service_test.go
// 워크스페이스 초대 서비스 단위 테스트
//
// 테스트 범위:
//   - SendInvitation: workspace-not-found, invitee-not-found, already-member,
//     duplicate-pending, 정상 생성
//   - ListWorkspaceInvitations: 전체 조회, 상태 필터 조회
//   - ListMyInvitations: 사용자별 PENDING 초대 조회
//   - AcceptInvitation: invitation-not-found, forbidden, not-pending, 정상 수락
//   - RejectInvitation: invitation-not-found, forbidden, not-pending, 정상 거절
//   - ListPendingApprovals: 상태별 전체 목록 조회
//   - ApproveInvitation: invitation-not-found, wrong-status, 정상 승인
//   - RejectInvitationByAdmin: invitation-not-found, wrong-status, 정상 거절

import (
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

func setupInvitationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Workspace{},
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.UserWorkspaceRole{},
		&model.WorkspaceInvitation{},
	))
	return db
}

// newTestInvitationService 는 SQLite in-memory DB 를 사용하는 WorkspaceInvitationService 를 생성합니다.
func newTestInvitationService(t *testing.T) (*WorkspaceInvitationService, *gorm.DB) {
	t.Helper()
	db := setupInvitationTestDB(t)
	svc := &WorkspaceInvitationService{
		db:                db,
		invitationRepo:    repository.NewWorkspaceInvitationRepository(db),
		workspaceRepo:     repository.NewWorkspaceRepository(db),
		userRepo:          repository.NewUserRepository(db),
		workspaceRoleRepo: repository.NewWorkspaceRoleRepository(db),
	}
	return svc, db
}

// createTestWorkspace 테스트용 워크스페이스를 DB에 삽입합니다.
func createTestWorkspace(t *testing.T, db *gorm.DB, name string) *model.Workspace {
	t.Helper()
	ws := &model.Workspace{Name: name}
	require.NoError(t, db.Create(ws).Error)
	return ws
}

// createInvTestUser 테스트용 사용자를 DB에 삽입합니다.
func createInvTestUser(t *testing.T, db *gorm.DB, kcID string) *model.User {
	t.Helper()
	require.NoError(t, db.Model(&model.User{}).Create(map[string]interface{}{
		"username": "user_" + kcID,
		"kc_id":    kcID,
	}).Error)
	var u model.User
	require.NoError(t, db.Where("kc_id = ?", kcID).First(&u).Error)
	return &u
}

// createTestInvitation 테스트용 초대를 DB에 직접 삽입합니다.
func createTestInvitation(t *testing.T, db *gorm.DB, wsID, inviterID, inviteeID uint, status model.InvitationStatus) *model.WorkspaceInvitation {
	t.Helper()
	inv := &model.WorkspaceInvitation{
		WorkspaceID:   wsID,
		InviterUserID: inviterID,
		InviteeUserID: inviteeID,
		Status:        status,
	}
	require.NoError(t, db.Model(&model.WorkspaceInvitation{}).Create(map[string]interface{}{
		"workspace_id":    inv.WorkspaceID,
		"inviter_user_id": inv.InviterUserID,
		"invitee_user_id": inv.InviteeUserID,
		"status":          string(inv.Status),
	}).Error)
	var created model.WorkspaceInvitation
	require.NoError(t, db.Where("workspace_id = ? AND invitee_user_id = ? AND status = ?", wsID, inviteeID, status).
		First(&created).Error)
	return &created
}

// ── SendInvitation 테스트 ─────────────────────────────────────────────────────

// TC-SI-01: 존재하지 않는 워크스페이스 → workspace not found 에러
func TestWorkspaceInvSendInvitation_WorkspaceNotFound(t *testing.T) {
	svc, _ := newTestInvitationService(t)

	_, err := svc.SendInvitation(99999, 1, 2, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace not found")
}

// TC-SI-02: 존재하지 않는 초대 대상 사용자 → invitee user not found 에러
func TestWorkspaceInvSendInvitation_InviteeNotFound(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-si-02")

	_, err := svc.SendInvitation(ws.ID, 1, 99999, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invitee user not found")
}

// TC-SI-03: 이미 워크스페이스 멤버인 사용자 → already a member 에러
func TestWorkspaceInvSendInvitation_AlreadyMember(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-si-03")
	inviter := createInvTestUser(t, db, "kc-inviter-si03")
	invitee := createInvTestUser(t, db, "kc-invitee-si03")

	// 워크스페이스 역할(RoleMaster + RoleSub) 생성 후 멤버 등록
	role := &model.RoleMaster{Name: "ws-role-si03"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO mcmp_role_subs (role_id, role_type) VALUES (?, ?)",
		role.ID, "workspace",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO mcmp_user_workspace_roles (user_id, workspace_id, role_id) VALUES (?, ?, ?)",
		invitee.ID, ws.ID, role.ID,
	).Error)

	_, err := svc.SendInvitation(ws.ID, inviter.ID, invitee.ID, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already a member")
}

// TC-SI-04: 이미 PENDING 초대가 존재하는 경우 → pending invitation already exists 에러
func TestWorkspaceInvSendInvitation_DuplicatePending(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-si-04")
	inviter := createInvTestUser(t, db, "kc-inviter-si04")
	invitee := createInvTestUser(t, db, "kc-invitee-si04")

	// 기존 PENDING 초대 삽입
	createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	_, err := svc.SendInvitation(ws.ID, inviter.ID, invitee.ID, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pending invitation already exists")
}

// TC-SI-05: 정상 초대 발송
func TestWorkspaceInvSendInvitation_Success(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-si-05")
	inviter := createInvTestUser(t, db, "kc-inviter-si05")
	invitee := createInvTestUser(t, db, "kc-invitee-si05")

	invitation, err := svc.SendInvitation(ws.ID, inviter.ID, invitee.ID, nil)

	require.NoError(t, err)
	require.NotNil(t, invitation)
	assert.Equal(t, model.InvitationStatusPending, invitation.Status)
	assert.Equal(t, ws.ID, invitation.WorkspaceID)
	assert.Equal(t, invitee.ID, invitation.InviteeUserID)
	assert.Greater(t, invitation.ID, uint(0))
}

// ── ListWorkspaceInvitations 테스트 ──────────────────────────────────────────

// TC-LWI-01: 전체 목록 조회 (status 필터 없음)
func TestWorkspaceInvListWorkspaceInvitations_All(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-lwi-01")
	inviter := createInvTestUser(t, db, "kc-inviter-lwi01")
	invitee1 := createInvTestUser(t, db, "kc-invitee-lwi01a")
	invitee2 := createInvTestUser(t, db, "kc-invitee-lwi01b")

	createTestInvitation(t, db, ws.ID, inviter.ID, invitee1.ID, model.InvitationStatusPending)
	createTestInvitation(t, db, ws.ID, inviter.ID, invitee2.ID, model.InvitationStatusAccepted)

	invitations, err := svc.ListWorkspaceInvitations(ws.ID, "")

	require.NoError(t, err)
	assert.Len(t, invitations, 2)
}

// TC-LWI-02: 상태 필터로 PENDING만 조회
func TestWorkspaceInvListWorkspaceInvitations_FilterByStatus(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-lwi-02")
	inviter := createInvTestUser(t, db, "kc-inviter-lwi02")
	invitee1 := createInvTestUser(t, db, "kc-invitee-lwi02a")
	invitee2 := createInvTestUser(t, db, "kc-invitee-lwi02b")

	createTestInvitation(t, db, ws.ID, inviter.ID, invitee1.ID, model.InvitationStatusPending)
	createTestInvitation(t, db, ws.ID, inviter.ID, invitee2.ID, model.InvitationStatusAccepted)

	invitations, err := svc.ListWorkspaceInvitations(ws.ID, string(model.InvitationStatusPending))

	require.NoError(t, err)
	assert.Len(t, invitations, 1)
	assert.Equal(t, model.InvitationStatusPending, invitations[0].Status)
}

// ── ListMyInvitations 테스트 ──────────────────────────────────────────────────

// TC-LMI-01: 내 PENDING 초대 목록 조회
func TestWorkspaceInvListMyInvitations_OnlyPending(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-lmi-01")
	inviter := createInvTestUser(t, db, "kc-inviter-lmi01")
	invitee := createInvTestUser(t, db, "kc-invitee-lmi01")

	createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)
	// ACCEPTED 상태는 포함되지 않아야 함
	ws2 := createTestWorkspace(t, db, "ws-lmi-01b")
	createTestInvitation(t, db, ws2.ID, inviter.ID, invitee.ID, model.InvitationStatusAccepted)

	invitations, err := svc.ListMyInvitations(invitee.ID)

	require.NoError(t, err)
	assert.Len(t, invitations, 1)
	assert.Equal(t, model.InvitationStatusPending, invitations[0].Status)
}

// ── AcceptInvitation 테스트 ──────────────────────────────────────────────────

// TC-AI-01: 존재하지 않는 초대 ID → invitation not found 에러
func TestWorkspaceInvAcceptInvitation_NotFound(t *testing.T) {
	svc, _ := newTestInvitationService(t)

	err := svc.AcceptInvitation(99999, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invitation not found")
}

// TC-AI-02: 초대받은 사용자가 아닌 경우 → forbidden 에러
func TestWorkspaceInvAcceptInvitation_Forbidden(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-ai-02")
	inviter := createInvTestUser(t, db, "kc-inviter-ai02")
	invitee := createInvTestUser(t, db, "kc-invitee-ai02")
	other := createInvTestUser(t, db, "kc-other-ai02")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	err := svc.AcceptInvitation(inv.ID, other.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

// TC-AI-03: PENDING 상태가 아닌 초대 수락 시도 → not in PENDING state 에러
func TestWorkspaceInvAcceptInvitation_NotPending(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-ai-03")
	inviter := createInvTestUser(t, db, "kc-inviter-ai03")
	invitee := createInvTestUser(t, db, "kc-invitee-ai03")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusAccepted)

	err := svc.AcceptInvitation(inv.ID, invitee.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in PENDING state")
}

// TC-AI-04: 정상 수락 (roleID 없음) → 상태 ACCEPTED 변경
func TestWorkspaceInvAcceptInvitation_Success_NoRole(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-ai-04")
	inviter := createInvTestUser(t, db, "kc-inviter-ai04")
	invitee := createInvTestUser(t, db, "kc-invitee-ai04")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	err := svc.AcceptInvitation(inv.ID, invitee.ID)

	require.NoError(t, err)

	var updated model.WorkspaceInvitation
	require.NoError(t, db.First(&updated, inv.ID).Error)
	assert.Equal(t, model.InvitationStatusAccepted, updated.Status)
}

// ── RejectInvitation 테스트 ──────────────────────────────────────────────────

// TC-RI-01: 존재하지 않는 초대 ID → invitation not found 에러
func TestWorkspaceInvRejectInvitation_NotFound(t *testing.T) {
	svc, _ := newTestInvitationService(t)

	err := svc.RejectInvitation(99999, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invitation not found")
}

// TC-RI-02: 초대받은 사용자가 아닌 경우 → forbidden 에러
func TestWorkspaceInvRejectInvitation_Forbidden(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-ri-02")
	inviter := createInvTestUser(t, db, "kc-inviter-ri02")
	invitee := createInvTestUser(t, db, "kc-invitee-ri02")
	other := createInvTestUser(t, db, "kc-other-ri02")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	err := svc.RejectInvitation(inv.ID, other.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

// TC-RI-03: PENDING 상태가 아닌 초대 거절 시도 → not in PENDING state 에러
func TestWorkspaceInvRejectInvitation_NotPending(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-ri-03")
	inviter := createInvTestUser(t, db, "kc-inviter-ri03")
	invitee := createInvTestUser(t, db, "kc-invitee-ri03")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusRejected)

	err := svc.RejectInvitation(inv.ID, invitee.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in PENDING state")
}

// TC-RI-04: 정상 거절 → 상태 REJECTED 변경
func TestWorkspaceInvRejectInvitation_Success(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-ri-04")
	inviter := createInvTestUser(t, db, "kc-inviter-ri04")
	invitee := createInvTestUser(t, db, "kc-invitee-ri04")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	err := svc.RejectInvitation(inv.ID, invitee.ID)

	require.NoError(t, err)

	var updated model.WorkspaceInvitation
	require.NoError(t, db.First(&updated, inv.ID).Error)
	assert.Equal(t, model.InvitationStatusRejected, updated.Status)
}

// ── ListPendingApprovals 테스트 ───────────────────────────────────────────────

// TC-LPA-01: 상태 필터 없이 전체 조회
func TestWorkspaceInvListPendingApprovals_All(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-lpa-01")
	inviter := createInvTestUser(t, db, "kc-inviter-lpa01")
	invitee1 := createInvTestUser(t, db, "kc-invitee-lpa01a")
	invitee2 := createInvTestUser(t, db, "kc-invitee-lpa01b")

	createTestInvitation(t, db, ws.ID, inviter.ID, invitee1.ID, model.InvitationStatusPendingApproval)
	createTestInvitation(t, db, ws.ID, inviter.ID, invitee2.ID, model.InvitationStatusRejected)

	invitations, err := svc.ListPendingApprovals("")

	require.NoError(t, err)
	assert.Len(t, invitations, 2)
}

// TC-LPA-02: PENDING_APPROVAL 상태로 필터링
func TestWorkspaceInvListPendingApprovals_FilterByStatus(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-lpa-02")
	inviter := createInvTestUser(t, db, "kc-inviter-lpa02")
	invitee1 := createInvTestUser(t, db, "kc-invitee-lpa02a")
	invitee2 := createInvTestUser(t, db, "kc-invitee-lpa02b")

	createTestInvitation(t, db, ws.ID, inviter.ID, invitee1.ID, model.InvitationStatusPendingApproval)
	createTestInvitation(t, db, ws.ID, inviter.ID, invitee2.ID, model.InvitationStatusRejected)

	invitations, err := svc.ListPendingApprovals(string(model.InvitationStatusPendingApproval))

	require.NoError(t, err)
	assert.Len(t, invitations, 1)
	assert.Equal(t, model.InvitationStatusPendingApproval, invitations[0].Status)
}

// ── ApproveInvitation 테스트 ─────────────────────────────────────────────────

// TC-APPI-01: 존재하지 않는 초대 ID → invitation not found 에러
func TestWorkspaceInvApproveInvitation_NotFound(t *testing.T) {
	svc, _ := newTestInvitationService(t)

	err := svc.ApproveInvitation(99999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invitation not found")
}

// TC-APPI-02: PENDING_APPROVAL 상태가 아닌 경우 → not in PENDING_APPROVAL state 에러
func TestWorkspaceInvApproveInvitation_WrongStatus(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-appi-02")
	inviter := createInvTestUser(t, db, "kc-inviter-appi02")
	invitee := createInvTestUser(t, db, "kc-invitee-appi02")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	err := svc.ApproveInvitation(inv.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in PENDING_APPROVAL state")
}

// TC-APPI-03: 정상 승인 (roleID 없음) → 상태 ACCEPTED 변경
func TestWorkspaceInvApproveInvitation_Success_NoRole(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-appi-03")
	inviter := createInvTestUser(t, db, "kc-inviter-appi03")
	invitee := createInvTestUser(t, db, "kc-invitee-appi03")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPendingApproval)

	err := svc.ApproveInvitation(inv.ID)

	require.NoError(t, err)

	var updated model.WorkspaceInvitation
	require.NoError(t, db.First(&updated, inv.ID).Error)
	assert.Equal(t, model.InvitationStatusAccepted, updated.Status)
}

// ── RejectInvitationByAdmin 테스트 ───────────────────────────────────────────

// TC-RBA-01: 존재하지 않는 초대 ID → invitation not found 에러
func TestWorkspaceInvRejectInvitationByAdmin_NotFound(t *testing.T) {
	svc, _ := newTestInvitationService(t)

	err := svc.RejectInvitationByAdmin(99999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invitation not found")
}

// TC-RBA-02: PENDING_APPROVAL 상태가 아닌 경우 → not in PENDING_APPROVAL state 에러
func TestWorkspaceInvRejectInvitationByAdmin_WrongStatus(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-rba-02")
	inviter := createInvTestUser(t, db, "kc-inviter-rba02")
	invitee := createInvTestUser(t, db, "kc-invitee-rba02")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPending)

	err := svc.RejectInvitationByAdmin(inv.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in PENDING_APPROVAL state")
}

// TC-RBA-03: 정상 관리자 거절 → 상태 REJECTED 변경
func TestWorkspaceInvRejectInvitationByAdmin_Success(t *testing.T) {
	svc, db := newTestInvitationService(t)
	ws := createTestWorkspace(t, db, "ws-rba-03")
	inviter := createInvTestUser(t, db, "kc-inviter-rba03")
	invitee := createInvTestUser(t, db, "kc-invitee-rba03")
	inv := createTestInvitation(t, db, ws.ID, inviter.ID, invitee.ID, model.InvitationStatusPendingApproval)

	err := svc.RejectInvitationByAdmin(inv.ID)

	require.NoError(t, err)

	var updated model.WorkspaceInvitation
	require.NoError(t, db.First(&updated, inv.ID).Error)
	assert.Equal(t, model.InvitationStatusRejected, updated.Status)
}
