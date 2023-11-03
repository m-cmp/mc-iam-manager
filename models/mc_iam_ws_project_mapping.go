package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// MCIamWsProjectMapping is used by pop to map your mc_iam_ws_project_mappings database table to your go code.
type MCIamWsProjectMapping struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	WsID      uuid.UUID       `json:"ws_id" db:"ws_id"`
	Ws        *MCIamWorkspace `belongs_to:"mc_iam_workspace"`
	ProjectID uuid.UUID       `json:"project_id" db:"project_id"`
	Project   *MCIamProject   `belongs_to:"mc_iam_project"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// String is not required by pop and may be deleted
func (m MCIamWsProjectMapping) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MCIamWsProjectMappings is not required by pop and may be deleted
type MCIamWsProjectMappings []MCIamWsProjectMapping

// String is not required by pop and may be deleted
func (m MCIamWsProjectMappings) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MCIamWsProjectMapping) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MCIamWsProjectMapping) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MCIamWsProjectMapping) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
