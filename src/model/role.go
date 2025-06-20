package model

import (
	"time"

	"github.com/m-cmp/mc-iam-manager/constants"
)

// RoleMaster 역할 마스터 모델 (DB 테이블: mcmp_role_masters)
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

// TableName RoleMaster의 테이블 이름을 반환
func (RoleMaster) TableName() string {
	return "mcmp_role_masters"
}

// RoleSub 역할 서브 모델 (DB 테이블: mcmp_role_sub)
type RoleSub struct {
	ID        uint                  `json:"id" gorm:"primaryKey;column:id"`
	RoleID    uint                  `json:"role_id" gorm:"column:role_id;not null"`
	RoleType  constants.IAMRoleType `json:"role_type" gorm:"column:role_type;size:50;not null"`
	CreatedAt time.Time             `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time             `json:"updated_at" gorm:"column:updated_at"`
}

// TableName RoleSub의 테이블 이름을 반환
func (RoleSub) TableName() string {
	return "mcmp_role_subs"
}

// UserPlatformRole 사용자-역할 매핑 모델 (DB 테이블: mcmp_user_platform_roles)
type UserPlatformRole struct {
	UserID    uint       `json:"user_id" gorm:"primaryKey;column:user_id"`
	RoleID    uint       `json:"role_id" gorm:"primaryKey;column:role_id"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	User      User       `json:"-" gorm:"foreignKey:UserID"`
	Role      RoleMaster `json:"-" gorm:"foreignKey:RoleID"`
}

// TableName UserPlatformRole의 테이블 이름을 반환
func (UserPlatformRole) TableName() string {
	return "mcmp_user_platform_roles"
}

// UserWorkspaceRole 사용자-워크스페이스-역할 매핑 모델 (DB 테이블: mcmp_user_workspace_roles)
// 사용자 기준으로 workspace와 role 을 표시 . workspace 기준으로는 WorkspaceWithProjects 를 사용
type UserWorkspaceRole struct {
	UserID        uint        `json:"user_id" gorm:"primaryKey;column:user_id"`
	WorkspaceID   uint        `json:"workspace_id" gorm:"primaryKey;column:workspace_id"`
	RoleID        uint        `json:"role_id" gorm:"primaryKey;column:role_id"`
	Username      string      `json:"username" gorm:"column:username"`
	WorkspaceName string      `json:"workspace_name" gorm:"column:workspace_name"`
	RoleName      string      `json:"role_name" gorm:"column:role_name"`
	CreatedAt     time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	User          *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Workspace     *Workspace  `json:"workspace,omitempty" gorm:"foreignKey:WorkspaceID;references:ID"`
	Role          *RoleMaster `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// TableName UserWorkspaceRole의 테이블 이름을 반환
func (UserWorkspaceRole) TableName() string {
	return "mcmp_user_workspace_roles"
}

// RoleMasterCspRoleMapping 역할 마스터-CSP 역할 매핑 모델 (DB 테이블: mcmp_role_csp_role_mapping)
// AUTH Method(OIDC, SAML, ...)
// CspRoleID는 mcmp_role_csps 테이블의 ID 참조
type RoleMasterCspRoleMapping struct {
	RoleID      uint                 `json:"roleId" gorm:"column:role_id;primaryKey;foreignKey:id;references:mcmp_role_masters"`
	AuthMethod  constants.AuthMethod `json:"auth_method" gorm:"column:auth_method;primaryKey"`
	CspRoleID   uint                 `json:"cspRoleId" gorm:"column:csp_role_id;primaryKey;foreignKey:id;references:mcmp_roles_csp"`
	Description string               `json:"description" gorm:"column:description"`
	CreatedAt   time.Time            `json:"createdAt" gorm:"column:created_at"`
	CspRole     *CspRole             `json:"cspRole" gorm:"foreignKey:CspRoleID;references:ID"`
}

// TableName RoleMasterCspRoleMapping의 테이블 이름을 반환
func (RoleMasterCspRoleMapping) TableName() string {
	return "mcmp_role_csp_role_mappings"
}
