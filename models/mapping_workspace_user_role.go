package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// MappingWorkspaceUserRole is used by pop to map your mapping_workspace_user_roles database table to your go code.
type MappingWorkspaceUserRole struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id" db:"workspace_id"`
	User        string    `json:"user_id" db:"user_id"`
	RoleID      uuid.UUID `json:"role_id" db:"role_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreateWorkspaceUserRoleMappingByNameRequest struct {
	WorkspaceName string
	User          string `json:"user"`
	RoleName      string `json:"roleName"`
}

type MappingWorkspaceUserRoleResponseWorkspace struct {
	Workspce  Workspace  `json:"workspace"`
	UserInfos []UserInfo `json:"userInfos"`
}

type MappingWorkspaceUserRoleResponseWorkspaces []MappingWorkspaceUserRoleResponseWorkspace

type MappingWorkspaceUserRoleResponseUser struct {
	Role                            Role                            `json:"role"`
	MappingWorkspaceProjectResponse MappingWorkspaceProjectResponse `json:"workspaceProject"`
}

type MappingWorkspaceUserRoleResponseUserArr []MappingWorkspaceUserRoleResponseUser

type UserInfo struct {
	User string `json:"user_id"`
	Role Role   `json:"role"`
}

// String is not required by pop and may be deleted
func (m MappingWorkspaceUserRole) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// MappingWorkspaceUserRoles is not required by pop and may be deleted
type MappingWorkspaceUserRoles []MappingWorkspaceUserRole

// String is not required by pop and may be deleted
func (m MappingWorkspaceUserRoles) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (m *MappingWorkspaceUserRole) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (m *MappingWorkspaceUserRole) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (m *MappingWorkspaceUserRole) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
