package repository

import (
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type CspRoleRepository struct {
	db *gorm.DB
}

func NewCspRoleRepository(db *gorm.DB) *CspRoleRepository {
	return &CspRoleRepository{
		db: db,
	}
}

// FindAll 모든 CSP 역할을 조회합니다.
func (r *CspRoleRepository) FindAll() ([]*model.CspRole, error) {
	var roles []*model.CspRole
	query := r.db.Find(&roles)
	if err := query.Error; err != nil {
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("FindAll SQL Query: %s", sql)
	log.Printf("FindAll SQL Args: %v", args)
	log.Printf("FindAll Result Count: %d", len(roles))

	return roles, nil
}

// Create 새로운 CSP 역할을 생성합니다.
func (r *CspRoleRepository) Create(role *model.CspRole) error {
	query := r.db.Create(role)
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Create SQL Query: %s", sql)
	log.Printf("Create SQL Args: %v", args)
	log.Printf("Create Created ID: %s", role.ID)

	return nil
}

// Update CSP 역할 정보를 수정합니다.
func (r *CspRoleRepository) Update(role *model.CspRole) error {
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

// Delete CSP 역할을 삭제합니다.
func (r *CspRoleRepository) Delete(id string) error {
	query := r.db.Delete(&model.CspRole{}, "id = ?", id)
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Delete SQL Query: %s", sql)
	log.Printf("Delete SQL Args: %v", args)
	log.Printf("Delete Affected Rows: %d", query.RowsAffected)

	return nil
}
