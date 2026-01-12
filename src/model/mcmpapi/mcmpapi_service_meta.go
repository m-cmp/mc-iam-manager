package mcmpapi

import "time"

// McmpApiServiceMeta stores version metadata for each service from _meta section
type McmpApiServiceMeta struct {
	ServiceName string    `gorm:"primaryKey;column:service_name;type:varchar(100)"`
	Version     string    `gorm:"column:version;type:varchar(50)"`
	Repository  string    `gorm:"column:repository;type:varchar(255)"`
	GeneratedAt time.Time `gorm:"column:generated_at"`
	SyncedAt    time.Time `gorm:"column:synced_at;autoUpdateTime"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName returns the table name for GORM
func (McmpApiServiceMeta) TableName() string {
	return "mcmp_api_service_meta"
}
