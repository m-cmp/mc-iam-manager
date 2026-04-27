package model

import "time"

// GroupPlatformRole 그룹-플랫폼 역할 매핑 (DB 테이블: mcmp_group_platform_roles)
// DB + Keycloak 이중 관리: 그룹에 realm role 매핑
type GroupPlatformRole struct {
	GroupID   uint      `gorm:"primaryKey;column:group_id" json:"group_id"`
	RoleID    uint      `gorm:"primaryKey;column:role_id" json:"role_id"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`

	Group *Organization `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Role  *RoleMaster   `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

func (GroupPlatformRole) TableName() string {
	return "mcmp_group_platform_roles"
}

// GroupWorkspaceRole 그룹-워크스페이스-역할 매핑 (DB 테이블: mcmp_group_workspace_roles)
// DB 전용 관리 (Keycloak 미사용)
type GroupWorkspaceRole struct {
	GroupID     uint      `gorm:"primaryKey;column:group_id" json:"group_id"`
	WorkspaceID uint      `gorm:"primaryKey;column:workspace_id" json:"workspace_id"`
	RoleID      uint      `gorm:"column:role_id;not null" json:"role_id"`
	CreatedAt   time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`

	Group     *Organization `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Workspace *Workspace    `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
	Role      *RoleMaster   `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

func (GroupWorkspaceRole) TableName() string {
	return "mcmp_group_workspace_roles"
}

// AssignGroupPlatformRoleRequest 그룹 플랫폼 역할 할당 요청
type AssignGroupPlatformRoleRequest struct {
	RoleID uint `json:"role_id" validate:"required"`
}

// AssignGroupWorkspaceRequest 그룹-워크스페이스 매핑 요청
type AssignGroupWorkspaceRequest struct {
	WorkspaceID uint `json:"workspace_id" validate:"required"`
	RoleID      uint `json:"role_id" validate:"required"`
}

// UpdateGroupWorkspaceRoleRequest 그룹 워크스페이스 역할 변경 요청
type UpdateGroupWorkspaceRoleRequest struct {
	RoleID uint `json:"role_id" validate:"required"`
}

// GroupPlatformRoleResponse 그룹 플랫폼 역할 목록 응답
type GroupPlatformRoleResponse struct {
	GroupID   uint      `json:"group_id"`
	GroupName string    `json:"group_name"`
	RoleID    uint      `json:"role_id"`
	RoleName  string    `json:"role_name"`
	CreatedAt time.Time `json:"created_at"`
}

// GroupWorkspaceRoleResponse 그룹 워크스페이스 역할 목록 응답
type GroupWorkspaceRoleResponse struct {
	GroupID       uint      `json:"group_id"`
	GroupName     string    `json:"group_name"`
	WorkspaceID   uint      `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	RoleID        uint      `json:"role_id"`
	RoleName      string    `json:"role_name"`
	CreatedAt     time.Time `json:"created_at"`
}

// AvailablePlatformRoleResponse 미할당 플랫폼 역할 응답
type AvailablePlatformRoleResponse struct {
	RoleID      uint   `json:"role_id"`
	RoleName    string `json:"role_name"`
	Description string `json:"description"`
}

// AssignUserGroupsRequest 사용자-그룹 할당 요청 (group_ids 사용)
type AssignUserGroupsRequest struct {
	GroupIDs []uint `json:"group_ids" validate:"required,min=1"`
}

// AssignGroupUsersRequest 그룹에 사용자 일괄 할당 요청 (group 입장)
type AssignGroupUsersRequest struct {
	UserIDs []uint `json:"user_ids" validate:"required,min=1"`
}

// PlatformRoleSimple 플랫폼 역할 간단 정보
type PlatformRoleSimple struct {
	RoleID      uint   `json:"role_id"`
	RoleName    string `json:"role_name"`
	Description string `json:"description"`
}

// EffectivePlatformRoleItem 유효 플랫폼 역할 항목 (직접 + 그룹 상속)
type EffectivePlatformRoleItem struct {
	RoleID      uint   `json:"role_id"`
	RoleName    string `json:"role_name"`
	Description string `json:"description"`
	Source      string `json:"source"` // "direct" 또는 "group:{groupName}"
}

// GroupAccessInfo 그룹 접근 정보 (그룹 기본 정보 + 할당된 역할 목록)
type GroupAccessInfo struct {
	GroupID   uint                 `json:"group_id"`
	GroupName string               `json:"group_name"`
	Roles     []PlatformRoleSimple `json:"roles"`
}

// UserAccessSummaryResponse 사용자 접근 권한 요약 응답
type UserAccessSummaryResponse struct {
	UserID      uint              `json:"user_id"`
	DirectRoles []PlatformRoleSimple `json:"direct_roles"`
	Groups      []GroupAccessInfo `json:"groups"`
}
