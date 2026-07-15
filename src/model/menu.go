package model

import (
	"time"
)

// Menu 메뉴 정보를 나타내는 구조체 (DB 테이블: mcmp_menu)
type Menu struct {
	ID          string    `json:"id" yaml:"id" gorm:"primaryKey;column:id"`
	ParentID    string    `json:"parentId,omitempty" yaml:"parentid" gorm:"column:parent_id"`
	DisplayName string    `json:"displayName" yaml:"displayname" gorm:"column:display_name;not null"`
	ResType     string    `json:"resType" yaml:"restype" gorm:"column:res_type;not null"`
	IsAction    bool      `json:"isAction" yaml:"isaction" gorm:"column:is_action;default:false"`
	Priority    uint      `json:"priority" yaml:"priority" gorm:"column:priority;not null"`
	MenuNumber  uint      `json:"menuNumber" yaml:"menunumber" gorm:"column:menu_number;not null"`
	ViewType         string `json:"viewType" yaml:"viewtype" gorm:"column:view_type;not null;default:local"`
	FrameworkService string `json:"frameworkService" yaml:"frameworkservice" gorm:"column:framework_service;not null;default:mc-web-console-front"`
	Path             string `json:"path" yaml:"path" gorm:"column:path;not null;default:''"`
	CreatedAt   time.Time `json:"-" yaml:"-" gorm:"column:created_at;autoCreateTime"` // GORM이 자동 처리
	UpdatedAt   time.Time `json:"-" yaml:"-" gorm:"column:updated_at;autoUpdateTime"` // GORM이 자동 처리
}

// TableName GORM에게 테이블 이름을 명시적으로 알려줌
func (Menu) TableName() string {
	return "mcmp_menus"
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

// RoleMenuMapping 역할-메뉴 매핑 (DB 테이블: mcmp_role_menu_mappings)
type RoleMenuMapping struct {
	ID        uint      `json:"id" gorm:"primaryKey;column:id"`
	RoleID    uint      `json:"role_id" gorm:"column:role_id;type:uint;not null"`         // 역할 ID
	MenuID    string    `json:"menu_id" gorm:"column:menu_id;type:varchar(100);not null"` // 메뉴 ID
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 테이블 이름 지정
func (RoleMenuMapping) TableName() string {
	return "mcmp_role_menu_mappings"
}

// CreateMenuResponse 메뉴 생성 응답 (메뉴 + 역할 매핑 포함)
type CreateMenuResponse struct {
	Menu         *Menu              `json:"menu"`
	RoleMappings []*RoleMenuMapping `json:"roleMappings"`
}

// RolePermissionEntry 역할 단위 권한 (permission.yaml / role-permission-backup 공통)
type RolePermissionEntry struct {
	Role       string   `json:"role" yaml:"role"`
	Menus      []string `json:"menus" yaml:"menus"`
	Operations []string `json:"operations" yaml:"operations"`
	Csps       []string `json:"csps" yaml:"csps"`
}

// RolePermissionBackup DB 역할 권한 백업 문서 (kind: role-permission-backup)
type RolePermissionBackup struct {
	Kind        string                `json:"kind" yaml:"kind"`
	BackupAt    string                `json:"backupAt" yaml:"backupAt"`
	Source      string                `json:"source" yaml:"source"`
	Sections    []string              `json:"sections" yaml:"sections"`
	Permissions []RolePermissionEntry `json:"permissions" yaml:"permissions"`
}

// RolePermissionRestoreResult restore 결과 요약
type RolePermissionRestoreResult struct {
	Mode           string `json:"mode"`
	RolesProcessed int    `json:"rolesProcessed"`
	MenusAdded     int    `json:"menusAdded"`
	MenusRemoved   int    `json:"menusRemoved"`
	Message        string `json:"message"`
}
