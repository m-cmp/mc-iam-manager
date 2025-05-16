package model

// McmpApiPermissionActionMapping API 권한-액션 매핑 모델
type McmpApiPermissionActionMapping struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	PermissionID string `json:"permission_id" gorm:"not null"`
	ActionID     int    `json:"action_id" gorm:"not null"`
	ActionName   string `json:"action_name" gorm:"not null"`
}

func (McmpApiPermissionActionMapping) TableName() string {
	return "mcmp_api_permission_action_mappings"
}
