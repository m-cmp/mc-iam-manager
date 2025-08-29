package mcmpapi // Change package name

import "time"

// McmpApiService represents the mcmp_api_services table. (Renamed)
type McmpApiService struct {
	Name      string    `gorm:"primaryKey;column:name;type:varchar(100)"` // Service name acts as PK
	Version   string    `gorm:"column:version;type:varchar(50)"`
	BaseURL   string    `gorm:"column:base_url;type:varchar(255)"`
	AuthType  string    `gorm:"column:auth_type;type:varchar(50)"`
	AuthUser  string    `gorm:"column:auth_user;type:varchar(100)"`
	AuthPass  string    `gorm:"column:auth_pass;type:varchar(255)"`      // Consider encryption for password
	IsActive  bool      `gorm:"column:is_active;default:false;not null"` // Added IsActive field
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName specifies the table name for McmpApiService. (Renamed receiver)
func (McmpApiService) TableName() string {
	return "mcmp_api_services" // Corrected table name
}
