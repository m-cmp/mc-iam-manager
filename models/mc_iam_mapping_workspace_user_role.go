package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

// MCIamMappingWorkspaceUserRole is used by pop to map your mc_iam_mapping_workspace_user_roles database table to your go code.
type MCIamMappingWorkspaceUserRole struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	RoleName    string    `json:"role_name" db:"role_name"`
	UserID      string    `json:"user_id" db:"user_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type MCIamMappingWorkspaceUserRoleListResponse struct {
	Workspace MCIamWorkspace            `json:"workspace"`
	Users     []UserRoleMappingResponse `json:"users"`
}

type MCIamMappingWorkspaceUserRoleUserResponse struct {
	RoleName  string         `json:"role"`
	Workspace MCIamWorkspace `json:"workspace"`
	Project   []MCIamProject `json:"projects"`
}

type MCIamMappingWorkspaceUserRoleUserResponses []MCIamMappingWorkspaceUserRoleUserResponse

type UserRoleMappingResponse struct {
	RoleName string `json:"role_name"`
	UserID   string `json:"user_id"`
}

// String is not required by pop and may be deleted
func (m MCIamMappingWorkspaceUserRole) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MCIamMappingWorkspaceUserRoles is not required by pop and may be deleted
type MCIamMappingWorkspaceUserRoles []MCIamMappingWorkspaceUserRole

// String is not required by pop and may be deleted
func (m MCIamMappingWorkspaceUserRoles) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MCIamMappingWorkspaceUserRole) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.StringIsPresent{Field: m.WorkspaceID, Name: "WorkspaceID"},
		&validators.StringIsPresent{Field: m.RoleName, Name: "RoleName"},
		&validators.StringIsPresent{Field: m.UserID, Name: "UserID"},
	), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MCIamMappingWorkspaceUserRole) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MCIamMappingWorkspaceUserRole) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
