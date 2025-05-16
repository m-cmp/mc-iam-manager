package repository

import (
	"errors"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type WorkspaceRoleRepository struct {
	db *gorm.DB
}

func NewWorkspaceRoleRepository(db *gorm.DB) *WorkspaceRoleRepository {
	return &WorkspaceRoleRepository{
		db: db,
	}
}

func (r *WorkspaceRoleRepository) List() ([]model.WorkspaceRole, error) {
	var roles []model.WorkspaceRole
	query := r.db.Find(&roles)
	if err := query.Error; err != nil {
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("List SQL Query: %s", sql)
	log.Printf("List SQL Args: %v", args)
	log.Printf("List Result Count: %d", len(roles))

	return roles, nil
}

func (r *WorkspaceRoleRepository) GetByID(id uint) (*model.WorkspaceRole, error) {
	var role model.WorkspaceRole
	query := r.db.First(&role, id)
	if err := query.Error; err != nil {
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetByID SQL Query: %s", sql)
	log.Printf("GetByID SQL Args: %v", args)

	return &role, nil
}

func (r *WorkspaceRoleRepository) Create(role *model.WorkspaceRole) error {
	query := r.db.Create(role)
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Create SQL Query: %s", sql)
	log.Printf("Create SQL Args: %v", args)
	log.Printf("Create Created ID: %d", role.ID)

	return nil
}

func (r *WorkspaceRoleRepository) Update(role *model.WorkspaceRole) error {
	query := r.db.Save(role)
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Update SQL Query: %s", sql)
	log.Printf("Update SQL Args: %v", args)
	log.Printf("Update Affected Rows: %d", query.RowsAffected)

	return nil
}

func (r *WorkspaceRoleRepository) Delete(id uint) error {
	query := r.db.Delete(&model.WorkspaceRole{}, id)
	if err := query.Error; err != nil {
		return err
	}
	if query.RowsAffected == 0 {
		return errors.New("workspace role not found or already deleted")
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Delete SQL Query: %s", sql)
	log.Printf("Delete SQL Args: %v", args)
	log.Printf("Delete Affected Rows: %d", query.RowsAffected)

	return nil
}

// AssignRoleToUser 사용자에게 워크스페이스 역할 할당 (mcmp_user_workspace_roles 테이블)
func (r *WorkspaceRoleRepository) AssignRoleToUser(userID, roleID, workspaceID uint) error {
	userWorkspaceRole := model.UserWorkspaceRole{
		UserID:          userID,
		WorkspaceID:     workspaceID,
		WorkspaceRoleID: roleID,
	}

	query := r.db.Create(&userWorkspaceRole)
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("AssignRoleToUser SQL Query: %s", sql)
	log.Printf("AssignRoleToUser SQL Args: %v", args)

	return nil
}

// RemoveRoleFromUser 사용자에게서 워크스페이스 역할 제거 (mcmp_user_workspace_roles 테이블)
func (r *WorkspaceRoleRepository) RemoveRoleFromUser(userID, roleID, workspaceID uint) error {
	query := r.db.Where("user_id = ? AND workspace_id = ? AND workspace_role_id = ?", userID, workspaceID, roleID).Delete(&model.UserWorkspaceRole{})
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("RemoveRoleFromUser SQL Query: %s", sql)
	log.Printf("RemoveRoleFromUser SQL Args: %v", args)
	log.Printf("RemoveRoleFromUser Affected Rows: %d", query.RowsAffected)

	return nil
}

// GetByName finds a workspace role by its name.
func (r *WorkspaceRoleRepository) GetByName(name string) (*model.WorkspaceRole, error) {
	var role model.WorkspaceRole
	query := r.db.Where("name = ?", name).First(&role)
	if err := query.Error; err != nil {
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetByName SQL Query: %s", sql)
	log.Printf("GetByName SQL Args: %v", args)

	return &role, nil
}
