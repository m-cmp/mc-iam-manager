package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// MCIamWsUserMapping is used by pop to map your mc_iam_ws_user_mappings database table to your go code.
type MCIamWsUserMapping struct {
	ID        uuid.UUID `json:"id" db:"id"`
	WsID      uuid.UUID `json:"workspaceId" db:"ws_id"`
	UserID    string    `json:"userId" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// String is not required by pop and may be deleted
func (m MCIamWsUserMapping) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MCIamWsUserMappings is not required by pop and may be deleted
type MCIamWsUserMappings []MCIamWsUserMapping

// String is not required by pop and may be deleted
func (m MCIamWsUserMappings) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MCIamWsUserMapping) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MCIamWsUserMapping) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MCIamWsUserMapping) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
