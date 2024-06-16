package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// MappingWorkspaceProject is used by pop to map your mapping_workspace_projects database table to your go code.
type MappingWorkspaceProject struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id" db:"workspace_id"`
	ProjectID   uuid.UUID `json:"project_id" db:"project_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
type MappingWorkspaceProjects []MappingWorkspaceProject

type MappingWorkspaceProjectsNameRequest struct {
	WorkspaceName string
	ProjectNames  []string `json:"projectNames"`
}

type MappingWorkspaceProjectsDeleteNameRequest struct {
	WorkspaceName string
	ProjectName   string
}

type MappingWorkspaceProjectsDeletIdRequest struct {
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
}

type MappingWorkspaceProjectsIdRequest struct {
	WorkspaceId string
	ProjectIds  []string `json:"projectIds"`
}

type MappingWorkspaceProjectResponse struct {
	Workspace Workspace `json:"workspace"`
	Projects  []Project `json:"projects"`
}
type MappingWorkspaceProjectResponses []MappingWorkspaceProjectResponse

// String is not required by pop and may be deleted
func (m MappingWorkspaceProject) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// String is not required by pop and may be deleted
func (m MappingWorkspaceProjects) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MappingWorkspaceProject) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MappingWorkspaceProject) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MappingWorkspaceProject) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
