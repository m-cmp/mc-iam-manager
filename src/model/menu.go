package model

import (
	"time"
)

// Menu 메뉴 정보를 나타내는 구조체 (DB 테이블: mcmp_menu)
type Menu struct {
	ID          string `json:"id" yaml:"id" gorm:"primaryKey;column:id"`
	ParentID    string `json:"parent_id,omitempty" yaml:"parentid" gorm:"column:parent_id"`
	DisplayName string `json:"display_name" yaml:"displayname" gorm:"column:display_name;not null"`
	ResType     string `json:"res_type" yaml:"restype" gorm:"column:res_type;not null"`
	IsAction    bool   `json:"is_action" yaml:"isaction" gorm:"column:is_action;default:false"`
	Priority    int    `json:"priority" yaml:"priority" gorm:"column:priority;not null"`
	MenuNumber  int    `json:"menu_number" yaml:"menunumber" gorm:"column:menu_number;not null"`
	// CreatedAt   time.Time `json:"-" yaml:"-" gorm:"column:created_at;autoCreateTime"` // GORM이 자동 처리
	// UpdatedAt   time.Time `json:"-" yaml:"-" gorm:"column:updated_at;autoUpdateTime"` // GORM이 자동 처리
}

// TableName GORM에게 테이블 이름을 명시적으로 알려줌
func (Menu) TableName() string {
	return "mcmp_menu"
}

// MenuTreeNode 메뉴 트리 구조를 위한 노드
type MenuTreeNode struct {
	Menu                     // Embed Menu fields directly
	Children []*MenuTreeNode `json:"children,omitempty"` // Slice of pointers to child nodes
}

// MenuData YAML 파일의 최상위 구조를 나타내는 구조체 (DB 전환 후에는 사용되지 않음)
// type MenuData struct {
// 	Menus []Menu `yaml:"menus"`
// }

// PlatformRoleMenuMapping 플랫폼 역할-메뉴 매핑 (DB 테이블: mcmp_platform_role_menu_mappings)
type PlatformRoleMenuMapping struct {
	ID           uint      `json:"id" gorm:"primaryKey;column:id"`
	PlatformRole string    `json:"platform_role" gorm:"column:platform_role;type:varchar(100);not null"` // 플랫폼 역할 이름
	MenuID       string    `json:"menu_id" gorm:"column:menu_id;type:varchar(100);not null"`             // 메뉴 ID
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 테이블 이름 지정
func (PlatformRoleMenuMapping) TableName() string {
	return "mcmp_platform_role_menu_mappings"
}
