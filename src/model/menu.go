package model

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

// MenuData YAML 파일의 최상위 구조를 나타내는 구조체 (DB 전환 후에는 사용되지 않음)
// type MenuData struct {
// 	Menus []Menu `yaml:"menus"`
// }
