package mcmpapi // Change package name

import "time"

// McmpApiAction represents the mcmp_api_actions table. (Renamed)
type McmpApiAction struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`                             // Auto-incrementing primary key
	ServiceName  string    `gorm:"column:service_name;type:varchar(100);not null;index"` // Foreign key reference (indexed)
	ActionName   string    `gorm:"column:action_name;type:varchar(100);not null"`
	Method       string    `gorm:"column:method;type:varchar(10);not null"`
	ResourcePath string    `gorm:"column:resource_path;type:varchar(500)"`
	Description  string    `gorm:"column:description;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
	// McmpApiService McmpApiService `gorm:"foreignKey:ServiceName;references:Name"` // Define relationship if needed (Use renamed service struct)
}

// TableName specifies the table name for McmpApiAction. (Renamed receiver)
func (McmpApiAction) TableName() string {
	return "mcmp_api_actions" // Corrected table name
}
