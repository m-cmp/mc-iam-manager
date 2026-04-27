package repository

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// WorkspaceInvitationRepository 워크스페이스 초대 레포지토리
type WorkspaceInvitationRepository struct {
	db *gorm.DB
}

// NewWorkspaceInvitationRepository 새 WorkspaceInvitationRepository 인스턴스 생성
func NewWorkspaceInvitationRepository(db *gorm.DB) *WorkspaceInvitationRepository {
	return &WorkspaceInvitationRepository{db: db}
}

// Create 초대 생성
func (r *WorkspaceInvitationRepository) Create(invitation *model.WorkspaceInvitation) error {
	return r.db.Create(invitation).Error
}

// FindByID ID로 초대 조회
func (r *WorkspaceInvitationRepository) FindByID(id uint) (*model.WorkspaceInvitation, error) {
	var invitation model.WorkspaceInvitation
	if err := r.db.First(&invitation, id).Error; err != nil {
		return nil, err
	}
	return &invitation, nil
}

// ListByWorkspace 워크스페이스 초대 목록 조회
func (r *WorkspaceInvitationRepository) ListByWorkspace(workspaceID uint, status string) ([]model.WorkspaceInvitation, error) {
	var invitations []model.WorkspaceInvitation
	query := r.db.Where("workspace_id = ?", workspaceID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

// ListByInvitee 초대받은 사용자의 초대 목록 조회
func (r *WorkspaceInvitationRepository) ListByInvitee(inviteeUserID uint) ([]model.WorkspaceInvitation, error) {
	var invitations []model.WorkspaceInvitation
	if err := r.db.Where("invitee_user_id = ? AND status = ?", inviteeUserID, model.InvitationStatusPending).
		Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

// ListByStatus 상태별 전체 초대 목록 조회 (관리자용)
func (r *WorkspaceInvitationRepository) ListByStatus(status string) ([]model.WorkspaceInvitation, error) {
	var invitations []model.WorkspaceInvitation
	query := r.db
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

// HasPendingInvitation 중복 PENDING 초대 확인
func (r *WorkspaceInvitationRepository) HasPendingInvitation(workspaceID, inviteeUserID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.WorkspaceInvitation{}).
		Where("workspace_id = ? AND invitee_user_id = ? AND status = ?",
			workspaceID, inviteeUserID, model.InvitationStatusPending).
		Count(&count).Error
	return count > 0, err
}

// UpdateStatus 초대 상태 업데이트
func (r *WorkspaceInvitationRepository) UpdateStatus(id uint, status model.InvitationStatus) error {
	return r.db.Model(&model.WorkspaceInvitation{}).
		Where("id = ?", id).
		Update("status", status).Error
}
