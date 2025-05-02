package mcmpapi

import "time"

// McmpApiPermissionActionMapping은 권한과 API 액션 간의 매핑을 나타냅니다.
type McmpApiPermissionActionMapping struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	PermissionID string    `gorm:"not null"`
	ActionID     uint      `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// TableName은 테이블 이름을 반환합니다.
func (McmpApiPermissionActionMapping) TableName() string {
	return "mcmp_api_permission_action_mappings"
}
