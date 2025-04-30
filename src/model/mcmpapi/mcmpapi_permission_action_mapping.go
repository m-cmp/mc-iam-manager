package mcmpapi

import "time"

// McmpApiPermissionActionMapping represents the mcmp_mciam_permission_action_mappings table.
type McmpApiPermissionActionMapping struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	PermissionID string    `gorm:"column:permission_id;type:varchar(255);not null;index"`
	ActionID     uint      `gorm:"column:action_id;not null;index"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
	// McmpApiAction McmpApiAction `gorm:"foreignKey:ActionID;references:ID"` // Define relationship if needed
}

// TableName specifies the table name for McmpApiPermissionActionMapping.
func (McmpApiPermissionActionMapping) TableName() string {
	return "mcmp_mciam_permission_action_mappings"
}
