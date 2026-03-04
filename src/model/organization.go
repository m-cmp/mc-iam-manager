package model

import "time"

// Organization 조직 모델 (DB 테이블: mcmp_organizations)
// 자체 조직관리 시스템 (Keycloak Groups 미사용)
// 계층적 Tree 구조: Self-Referencing FK (parent_id)
type Organization struct {
	ID               uint          `gorm:"primaryKey" json:"id"`
	ParentID         *uint         `gorm:"column:parent_id" json:"parent_id,omitempty"`
	OrganizationCode string        `gorm:"size:20;not null;unique" json:"organization_code"`
	Name             string        `gorm:"size:255;not null" json:"name"`
	Description      string        `gorm:"size:1000" json:"description,omitempty"`
	CreatedAt        time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time     `gorm:"autoUpdateTime" json:"updated_at"`

	// 관계 정의 (API 응답 전용 - 필요 시 Preload)
	Parent   *Organization  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Organization `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Users    []*User        `gorm:"many2many:mcmp_user_organizations;foreignKey:ID;joinForeignKey:organization_id;References:ID;joinReferences:user_id" json:"users,omitempty"`
}

// TableName Organization의 테이블 이름을 지정합니다
func (Organization) TableName() string {
	return "mcmp_organizations"
}

// OrganizationTree Tree 구조 응답용 (재귀)
type OrganizationTree struct {
	ID               uint               `json:"id"`
	ParentID         *uint              `json:"parent_id,omitempty"`
	OrganizationCode string             `json:"organization_code"`
	Name             string             `json:"name"`
	Description      string             `json:"description,omitempty"`
	Level            int                `json:"level"`
	Path             string             `json:"path"` // 예: "/조직A/개발팀"
	UserCount        int                `json:"user_count"`
	Children         []OrganizationTree `json:"children,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// UserOrganization 사용자-조직 M:N 매핑 모델 (DB 테이블: mcmp_user_organizations)
// 한 사용자가 여러 조직에 소속 가능 (다중 소속)
type UserOrganization struct {
	UserID         uint      `gorm:"primaryKey;column:user_id" json:"user_id"`
	OrganizationID uint      `gorm:"primaryKey;column:organization_id" json:"organization_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`

	// 관계 (JOIN 시 사용)
	User         *User         `gorm:"foreignKey:UserID" json:"-"`
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

// TableName UserOrganization의 테이블 이름을 지정합니다
func (UserOrganization) TableName() string {
	return "mcmp_user_organizations"
}

// CreateOrganizationRequest 조직 생성 요청
type CreateOrganizationRequest struct {
	Name             string `json:"name" validate:"required,max=255"`
	Description      string `json:"description" validate:"max=1000"`
	ParentID         *uint  `json:"parent_id"`         // nil = 최상위 조직
	OrganizationCode string `json:"organization_code"` // 비어있으면 자동 생성
}

// UpdateOrganizationRequest 조직 수정 요청
type UpdateOrganizationRequest struct {
	Name             string `json:"name" validate:"max=255"`
	Description      string `json:"description" validate:"max=1000"`
	ParentID         *uint  `json:"parent_id"`         // 부모 변경 시 입력
	OrganizationCode string `json:"organization_code"` // 코드 수정 시 입력
}

// OrganizationSeedItem YAML 시드 데이터 항목
type OrganizationSeedItem struct {
	OrganizationCode string                 `yaml:"organization_code"`
	Name             string                 `yaml:"name"`
	Description      string                 `yaml:"description"`
	Children         []OrganizationSeedItem `yaml:"children,omitempty"`
}

// OrganizationSeedData YAML 최상위 구조
type OrganizationSeedData struct {
	Organizations []OrganizationSeedItem `yaml:"organizations"`
}

// AssignUserOrganizationsRequest 사용자-조직 할당 요청
type AssignUserOrganizationsRequest struct {
	OrganizationIDs []uint `json:"organization_ids" validate:"required,min=1"`
}

// OrganizationResponse 조직 단건 응답
type OrganizationResponse struct {
	ID               uint      `json:"id"`
	ParentID         *uint     `json:"parent_id,omitempty"`
	OrganizationCode string    `json:"organization_code"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Path             string    `json:"path"` // 예: "/조직A/개발팀"
	UserCount        int       `json:"user_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
