package model

import "time"

// User 사용자 모델 (DB 테이블: mcmp_users)
type User struct {
	// Keycloak에서 가져오는 정보 (DB에 직접 저장되지 않을 수 있음, Repository에서 처리)
	ID        string `json:"id" gorm:"-"` // DB에는 저장 안 함 (Keycloak ID 사용)
	Username  string `json:"username" gorm:"-"`
	Email     string `json:"email" gorm:"-"`
	FirstName string `json:"firstName,omitempty" gorm:"-"`
	LastName  string `json:"lastName,omitempty" gorm:"-"`
	Enabled   bool   `json:"enabled" gorm:"-"`

	// DB에 저장되는 정보 (mcmp_users 테이블)
	DbId        uint      `json:"-" gorm:"primaryKey;column:id"`                  // DB 내부 ID
	KcId        string    `json:"-" gorm:"column:kc_id;size:255;not null;unique"` // Keycloak User ID 저장용 컬럼 추가 필요
	Description string    `json:"description,omitempty" gorm:"column:description;size:1000"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// 관계 정의
	PlatformRoles  []*PlatformRole  `json:"platform_roles,omitempty" gorm:"many2many:mcmp_user_platform_roles;foreignKey:DbId;joinForeignKey:user_id;References:ID;joinReferences:platform_role_id"`
	WorkspaceRoles []*WorkspaceRole `json:"workspace_roles,omitempty" gorm:"many2many:mcmp_user_workspace_roles;foreignKey:DbId;joinForeignKey:user_id;References:ID;joinReferences:workspace_role_id"`
}

// TableName User의 테이블 이름을 지정합니다
func (User) TableName() string {
	return "mcmp_users"
}
