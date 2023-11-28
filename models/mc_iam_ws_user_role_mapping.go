package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// MCIamWsUserRoleMapping is used by pop to map your mc_iam_ws_user_role_mappings database table to your go code.
type MCIamWsUserRoleMapping struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	WsID      uuid.UUID       `json:"ws_id" db:"ws_id"`
	Ws        *MCIamWorkspace `belongs_to:"mc_iam_workspaces"`
	UserID    string          `json:"user_id" db:"user_id"`
	RoleID    uuid.UUID       `json:"role_id" db:"role_id"`
	Role      *MCIamRole      `belongs_to:"mc_iam_role"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// String is not required by pop and may be deleted
func (m MCIamWsUserRoleMapping) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MCIamWsUserRoleMappings is not required by pop and may be deleted
type MCIamWsUserRoleMappings []MCIamWsUserRoleMapping

// String is not required by pop and may be deleted
func (m MCIamWsUserRoleMappings) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MCIamWsUserRoleMapping) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MCIamWsUserRoleMapping) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MCIamWsUserRoleMapping) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
