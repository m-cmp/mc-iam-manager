package repository

import (
	"errors"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type PlatformRoleRepository struct {
	db *gorm.DB
}

func NewPlatformRoleRepository(db *gorm.DB) *PlatformRoleRepository {
	return &PlatformRoleRepository{
		db: db,
	}
}

func (r *PlatformRoleRepository) List() ([]model.PlatformRole, error) {
	var roles []model.PlatformRole
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

func (r *PlatformRoleRepository) GetByID(id uint) (*model.PlatformRole, error) {
	var role model.PlatformRole
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

func (r *PlatformRoleRepository) Create(role *model.PlatformRole) error {
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

func (r *PlatformRoleRepository) Update(role *model.PlatformRole) error {
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

func (r *PlatformRoleRepository) Delete(id uint) error {
	query := r.db.Delete(&model.PlatformRole{}, id)
	if err := query.Error; err != nil {
		return err
	}
	if query.RowsAffected == 0 {
		return errors.New("platform role not found or already deleted")
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Delete SQL Query: %s", sql)
	log.Printf("Delete SQL Args: %v", args)
	log.Printf("Delete Affected Rows: %d", query.RowsAffected)

	return nil
}

// GetByName finds a platform role by its name.
func (r *PlatformRoleRepository) GetByName(name string) (*model.PlatformRole, error) {
	var role model.PlatformRole
	query := r.db.Where("name = ?", name).First(&role)
	if err := query.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Consider returning a specific error like ErrPlatformRoleNotFound
			return nil, errors.New("platform role not found")
		}
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetByName SQL Query: %s", sql)
	log.Printf("GetByName SQL Args: %v", args)

	return &role, nil
}

// DB returns the underlying gorm DB instance (Helper for sync function)
func (r *PlatformRoleRepository) DB() *gorm.DB {
	return r.db
}
