package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

// WorkspaceUserRoleMapping is used by pop to map your workspace_user_role_mappings database table to your go code.
type WorkspaceUserRoleMapping struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id" db:"workspace_id"`
	UserID      string    `json:"user_id" db:"user_id"`
	RoleID      uuid.UUID `json:"role_id" db:"role_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// String is not required by pop and may be deleted
func (w WorkspaceUserRoleMapping) String() string {
	jw, _ := json.Marshal(w)
	return string(jw)
}

// WorkspaceUserRoleMappings is not required by pop and may be deleted
type WorkspaceUserRoleMappings []WorkspaceUserRoleMapping

// String is not required by pop and may be deleted
func (w WorkspaceUserRoleMappings) String() string {
	jw, _ := json.Marshal(w)
	return string(jw)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (w *WorkspaceUserRoleMapping) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.StringIsPresent{Field: w.UserID, Name: "UserID"},
	), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (w *WorkspaceUserRoleMapping) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (w *WorkspaceUserRoleMapping) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
