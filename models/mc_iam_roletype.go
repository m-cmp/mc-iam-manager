package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

// MCIamRoletype is used by pop to map your mc_iam_roletypes database table to your go code.
type MCIamRoletype struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Type      string    `json:"type" db:"type"`
	RoleID    string    `json:"role_id" db:"role_id"`
	RoleName  string    `json:"role_name" db:"role_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// String is not required by pop and may be deleted
func (m MCIamRoletype) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MCIamRoletypes is not required by pop and may be deleted
type MCIamRoletypes []MCIamRoletype

// String is not required by pop and may be deleted
func (m MCIamRoletypes) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MCIamRoletype) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.StringIsPresent{Field: m.Type, Name: "Type"},
		&validators.StringIsPresent{Field: m.RoleID, Name: "RoleID"},
		&validators.StringIsPresent{Field: m.RoleName, Name: "RoleName"},
	), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MCIamRoletype) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MCIamRoletype) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
