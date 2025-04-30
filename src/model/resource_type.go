package model

import "time"

// ResourceType represents a manageable resource category within a specific framework.
type ResourceType struct {
	FrameworkID string    `gorm:"primaryKey;type:varchar(100);not null" json:"frameworkId"` // Identifier of the framework (e.g., "mc-iam-manager", "mc-infra-manager")
	ID          string    `gorm:"primaryKey;type:varchar(100);not null" json:"id"`          // Unique identifier within the framework (e.g., "workspace", "vm")
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`                   // Display name (e.g., "Workspace", "Virtual Machine")
	Description string    `gorm:"type:varchar(1000)" json:"description"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updatedAt"`
}

// TableName specifies the table name for ResourceType model
func (ResourceType) TableName() string {
	return "mcmp_resource_types"
}
