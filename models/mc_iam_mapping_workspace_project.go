package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

// MCIamMappingWorkspaceProject is used by pop to map your mc_iam_mapping_workspace_projects database table to your go code.
type MCIamMappingWorkspaceProject struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	WorkspaceID string          `json:"workspace_id" db:"workspace_id"`
	Workspace   *MCIamWorkspace `belongs_to:"mc_iam_workspaces"`
	ProjectID   string          `json:"project_id" db:"project_id"`
	Project     *MCIamProject   `belongs_to:"mc_iam_projects"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// String is not required by pop and may be deleted
func (m MCIamMappingWorkspaceProject) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MCIamMappingWorkspaceProjects is not required by pop and may be deleted
type MCIamMappingWorkspaceProjects []MCIamMappingWorkspaceProject

// String is not required by pop and may be deleted
func (m MCIamMappingWorkspaceProjects) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MCIamMappingWorkspaceProject) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.StringIsPresent{Field: m.WorkspaceID, Name: "WorkspaceID"},
		&validators.StringIsPresent{Field: m.ProjectID, Name: "ProjectID"},
	), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MCIamMappingWorkspaceProject) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MCIamMappingWorkspaceProject) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
