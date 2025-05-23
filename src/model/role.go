package model

import "time"

// RoleMaster 역할 마스터 모델 (DB 테이블: mcmp_role_master)
type RoleMaster struct {
	ID          uint         `json:"id" gorm:"primaryKey;column:id"`
	ParentID    *uint        `json:"parent_id" gorm:"column:parent_id"`
	Name        string       `json:"name" gorm:"column:name;size:255;not null;unique"`
	Description string       `json:"description" gorm:"column:description;size:1000"`
	Predefined  bool         `json:"predefined" gorm:"column:predefined;not null;default:false"`
	CreatedAt   time.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Parent      *RoleMaster  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children    []RoleMaster `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	RoleSubs    []RoleSub    `json:"role_subs,omitempty" gorm:"foreignKey:RoleID"`
}

// TableName RoleMaster의 테이블 이름을 지정합니다
func (RoleMaster) TableName() string {
	return "mcmp_role_master"
}

// RoleSub 역할 서브 모델 (DB 테이블: mcmp_role_sub)
type RoleSub struct {
	ID        uint       `json:"id" gorm:"primaryKey;column:id"`
	RoleID    uint       `json:"role_id" gorm:"column:role_id;not null"`
	RoleType  string     `json:"role_type" gorm:"column:role_type;size:50;not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	Role      RoleMaster `json:"-" gorm:"foreignKey:RoleID"`
}

// TableName RoleSub의 테이블 이름을 지정합니다
func (RoleSub) TableName() string {
	return "mcmp_role_sub"
}

// UserRole 사용자-역할 매핑 모델 (DB 테이블: mcmp_user_platform_roles)
type UserPlatformRole struct {
	UserID    uint       `json:"user_id" gorm:"primaryKey;column:user_id"`
	RoleID    uint       `json:"role_id" gorm:"primaryKey;column:role_id"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	User      User       `json:"-" gorm:"foreignKey:UserID"`
	Role      RoleMaster `json:"-" gorm:"foreignKey:RoleID"`
}

// TableName UserRole의 테이블 이름을 지정합니다
func (UserPlatformRole) TableName() string {
	return "mcmp_user_platform_roles"
}

// UserWorkspaceRole 사용자-워크스페이스-역할 매핑 모델 (DB 테이블: mcmp_user_workspace_roles)
type UserWorkspaceRole struct {
	UserID      uint        `json:"user_id" gorm:"primaryKey;column:user_id"`
	WorkspaceID uint        `json:"workspace_id" gorm:"primaryKey;column:workspace_id"`
	RoleID      uint        `json:"role_id" gorm:"primaryKey;column:role_id"`
	CreatedAt   time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	User        *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Workspace   *Workspace  `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID"`
	Role        *RoleMaster `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// TableName UserWorkspaceRole의 테이블 이름을 지정합니다
func (UserWorkspaceRole) TableName() string {
	return "mcmp_user_workspace_roles"
}

// UserWorkspaceRoleResponse 사용자-워크스페이스-역할 정보를 담는 응답 구조체
type UserWorkspaceRoleResponse struct {
	UserID        uint      `json:"user_id"`
	Username      string    `json:"username"`
	WorkspaceID   uint      `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	RoleID        uint      `json:"role_id"`
	RoleName      string    `json:"role_name"`
	CreatedAt     time.Time `json:"created_at"`
}

// RoleType 상수 정의
const (
	RoleTypePlatform  = "platform"
	RoleTypeWorkspace = "workspace"
)

// AssignRoleRequest 역할 할당 요청 구조체
type AssignRoleRequest struct {
	Username    string `json:"username"`
	RoleName    string `json:"roleName"`
	WorkspaceID uint   `json:"workspaceId"`
}
