package model

// Menu 메뉴 정보를 나타내는 구조체
type Menu struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent_id,omitempty"`
	DisplayName string `json:"display_name"`
	ResType     string `json:"res_type"`
	IsAction    bool   `json:"is_action"`
	Priority    int    `json:"priority"`
	MenuNumber  int    `json:"menu_number"`
}
