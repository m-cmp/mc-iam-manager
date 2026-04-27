package model

import "time"

// UserStatus 사용자 계정 상태
type UserStatus string

const (
	UserStatusActive              UserStatus = "ACTIVE"
	UserStatusInactive            UserStatus = "INACTIVE"
	UserStatusWithdrawalRequested UserStatus = "WITHDRAWAL_REQUESTED"
	UserStatusWithdrawn           UserStatus = "WITHDRAWN"
)

// User 사용자 모델 (DB 테이블: mcmp_users)
type User struct {
	// Keycloak 정보
	Username     string `json:"username" gorm:"column:username;size:255;not null;unique"` // Keep Username mapped to DB
	Email        string `json:"email" gorm:"-"`                                           // Ignore Email for DB
	FirstName    string `json:"firstName,omitempty" gorm:"-"`                             // Ignore FirstName for DB
	LastName     string `json:"lastName,omitempty" gorm:"-"`                              // Ignore LastName for DB
	Enabled      bool   `json:"enabled" gorm:"-"`                                        // Enabled status managed by Keycloak
	Organization string `json:"organization,omitempty" gorm:"-"`                          // Organization stored in Keycloak attributes

	// DB에 저장되는 정보 (mcmp_users 테이블)
	ID          uint       `json:"id" gorm:"primaryKey;column:id"`                              // DB Primary Key (Renamed from DbId)
	KcId        string     `json:"kc_id" gorm:"column:kc_id;size:255;not null;unique"`          // Keycloak User ID
	Status      UserStatus `json:"status" gorm:"column:status;size:50;default:'ACTIVE'"`        // 사용자 계정 상태
	Description string     `json:"description,omitempty" gorm:"column:description;size:1000"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// 관계 정의
	PlatformRoles  []*RoleMaster `json:"platform_roles,omitempty" gorm:"many2many:mcmp_user_platform_roles;foreignKey:ID;joinForeignKey:user_id;References:ID;joinReferences:role_id;joinTable:mcmp_user_platform_roles;where:role_type='platform'"`
	WorkspaceRoles []*RoleMaster `json:"workspace_roles,omitempty" gorm:"many2many:mcmp_user_workspace_roles;foreignKey:ID;joinForeignKey:user_id;References:ID;joinReferences:role_id;joinTable:mcmp_user_workspace_roles;where:role_type='workspace'"`
}

// TableName User의 테이블 이름을 지정합니다
func (User) TableName() string {
	return "mcmp_users"
}
