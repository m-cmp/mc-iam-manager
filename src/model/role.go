package model

import "time"

// PlatformRole 플랫폼 역할 모델 (DB 테이블: mcmp_platform_roles)
type PlatformRole struct {
	ID          uint      `json:"id" gorm:"primaryKey;column:id"`
	Name        string    `json:"name" gorm:"column:name;size:255;not null;unique"`
	Description string    `json:"description" gorm:"column:description;size:1000"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName PlatformRole의 테이블 이름을 지정합니다
func (PlatformRole) TableName() string {
	return "mcmp_platform_roles"
}

// WorkspaceRole 워크스페이스 역할 모델 (DB 테이블: mcmp_workspace_roles)
// PlatformRole처럼 독립적으로 정의되어야 함. WorkspaceID 제거.
type WorkspaceRole struct {
	ID          uint      `json:"id" gorm:"primaryKey;column:id"`
	Name        string    `json:"name" gorm:"column:name;size:255;not null;unique"` // 이름은 고유해야 함
	Description string    `json:"description" gorm:"column:description;size:1000"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName WorkspaceRole의 테이블 이름을 지정합니다
func (WorkspaceRole) TableName() string {
	// GORM doesn't directly support composite unique constraints via tags easily,
	// ensure it's defined in the migration.
	return "mcmp_workspace_roles"
}

// UserPlatformRole 사용자-플랫폼 역할 매핑 모델 (DB 테이블: mcmp_user_platform_roles)
type UserPlatformRole struct {
	UserID         uint         `json:"user_id" gorm:"primaryKey;column:user_id"`
	PlatformRoleID uint         `json:"platform_role_id" gorm:"primaryKey;column:platform_role_id"`
	CreatedAt      time.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	User           User         `json:"-" gorm:"foreignKey:UserID"`         // Belongs To User
	PlatformRole   PlatformRole `json:"-" gorm:"foreignKey:PlatformRoleID"` // Belongs To PlatformRole
}

// TableName UserPlatformRole의 테이블 이름을 지정합니다
func (UserPlatformRole) TableName() string {
	return "mcmp_user_platform_roles"
}

// UserWorkspaceRole 사용자-워크스페이스-역할 매핑 모델 (DB 테이블: mcmp_user_workspace_roles)
// WorkspaceID 추가 및 복합 기본 키 설정 필요
type UserWorkspaceRole struct {
	UserID          uint          `json:"user_id" gorm:"primaryKey;column:user_id;autoIncrement:false"`                     // 복합 키의 일부
	WorkspaceID     uint          `json:"workspace_id" gorm:"primaryKey;column:workspace_id;autoIncrement:false"`           // 복합 키의 일부, FK
	WorkspaceRoleID uint          `json:"workspace_role_id" gorm:"primaryKey;column:workspace_role_id;autoIncrement:false"` // 복합 키의 일부, FK
	CreatedAt       time.Time     `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	User            User          `json:"-" gorm:"foreignKey:UserID;references:ID"` // Corrected references to ID
	Workspace       Workspace     `json:"-" gorm:"foreignKey:WorkspaceID"`          // Belongs To Workspace
	WorkspaceRole   WorkspaceRole `json:"-" gorm:"foreignKey:WorkspaceRoleID"`      // Belongs To WorkspaceRole
}

// TableName UserWorkspaceRole의 테이블 이름을 지정합니다
func (UserWorkspaceRole) TableName() string {
	return "mcmp_user_workspace_roles"
}
