package repository

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/util"
	"gopkg.in/yaml.v3" // Use v3 as intended
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // For Upsert/OnConflict
)

var (
	ErrMenuNotFound = errors.New("menu not found")
	// ErrMenuAlreadyExists는 GORM의 기본 에러 처리(예: 제약 조건 위반)로 대체될 수 있음
)

// MenuRepository 데이터베이스에서 메뉴 데이터를 관리
type MenuRepository struct {
	db *gorm.DB
}

// NewMenuRepository 새 MenuRepository 인스턴스 생성
func NewMenuRepository(db *gorm.DB) *MenuRepository {
	// AutoMigrate the required tables
	err := db.AutoMigrate(
		&model.Menu{},
		&model.RoleMenuMapping{},
		&model.MciamPermission{},
		&model.MciamRoleMciamPermission{},
		&model.ResourceType{},
	)
	if err != nil {
		log.Printf("Failed to auto migrate tables: %v", err)
	}
	return &MenuRepository{db: db}
}

// GetMenus 데이터베이스에서 모든 메뉴 조회
func (r *MenuRepository) GetMenus(req *model.MenuFilterRequest) ([]*model.Menu, error) {
	var menus []*model.Menu
	query := r.db.Model(&model.Menu{})

	if len(req.MenuNames) > 0 {
		query = query.Where("name IN ?", req.MenuNames)
	}
	if len(req.MenuIDs) > 0 {
		query = query.Where("id IN ?", req.MenuIDs)
	}

	// GORM은 기본적으로 UpdatedAt DESC 정렬을 시도할 수 있으므로 명시적 정렬 추가
	if err := query.Order("priority asc, menu_number asc").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

// GetByID 메뉴 ID로 데이터베이스에서 조회
func (r *MenuRepository) FindMenuByID(id *string) (*model.Menu, error) {
	var menu model.Menu
	if err := r.db.Where("id = ?", id).First(&menu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMenuNotFound // 사용자 정의 에러 반환 또는 nil, nil 반환
		}
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := r.db.Statement.SQL.String()
	args := r.db.Statement.Vars
	log.Printf("GetByID SQL Query: %s", sql)
	log.Printf("GetByID SQL Args: %v", args)

	return &menu, nil
}

func (r *MenuRepository) FindParentIDs(menuIDs []*string) ([]*string, error) {
	var parentIDs []*string
	err := r.db.Model(&model.Menu{}).
		Where("id IN ? AND parent_id IS NOT NULL", menuIDs).
		Distinct().
		Pluck("parent_id", &parentIDs).Error
	return parentIDs, err
}

// Create 새 메뉴를 데이터베이스에 생성
func (r *MenuRepository) CreateMenu(req *model.CreateMenuRequest) error {
	priorityInt, err := util.StringToUint(req.Priority)
	if err != nil {
		return err
	}
	menuNumberInt, err := util.StringToUint(req.MenuNumber)
	if err != nil {
		return err
	}
	menu := &model.Menu{
		ID:          req.ID,
		ParentID:    req.ParentID,
		DisplayName: req.DisplayName,
		ResType:     req.ResType,
		IsAction:    req.IsAction,
		Priority:    priorityInt,
		MenuNumber:  menuNumberInt,
	}
	return r.db.Create(menu).Error
}

// Update 기존 메뉴를 데이터베이스에서 부분 업데이트
func (r *MenuRepository) UpdateMenu(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update") // 업데이트할 필드가 없음
	}

	// GORM의 Updates 메서드는 map[string]interface{}를 사용하여 지정된 필드만 업데이트
	result := r.db.Model(&model.Menu{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMenuNotFound // 업데이트 대상 레코드가 없음
	}

	// SQL 쿼리 로깅
	sql := result.Statement.SQL.String()
	args := result.Statement.Vars
	log.Printf("Update SQL Query: %s", sql)
	log.Printf("Update SQL Args: %v", args)
	log.Printf("Update Affected Rows: %d", result.RowsAffected)

	return nil
}

// Delete 메뉴를 데이터베이스에서 삭제
func (r *MenuRepository) DeleteMenu(id string) error {
	// GORM의 Delete는 삭제된 행 수를 반환
	result := r.db.Where("id = ?", id).Delete(&model.Menu{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMenuNotFound
	}

	// SQL 쿼리 로깅
	sql := result.Statement.SQL.String()
	args := result.Statement.Vars
	log.Printf("Delete SQL Query: %s", sql)
	log.Printf("Delete SQL Args: %v", args)
	log.Printf("Delete Affected Rows: %d", result.RowsAffected)

	return nil
}

// LoadMenusFromYAML YAML 파일에서 메뉴 데이터를 로드 (내부 헬퍼)
func (r *MenuRepository) LoadMenusFromYAML(filePath string) ([]model.Menu, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 파일 없으면 빈 목록 반환 (에러 아님)
		return []model.Menu{}, nil
	}

	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading menu file %s: %w", filePath, err)
	}

	var menuData struct { // 임시 구조체 사용 (model.MenuData 주석 처리됨)
		Menus []model.Menu `yaml:"menus"`
	}
	err = yaml.Unmarshal(yamlFile, &menuData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling menu file %s: %w", filePath, err)
	}
	return menuData.Menus, nil
}

// UpsertMenus 메뉴 목록을 DB에 Upsert (있으면 업데이트, 없으면 생성)
// 외부에서 메뉴 목록을 받아 처리. 트랜잭션 내에서 실행하고 제약 조건 검사를 지연시킴.
func (r *MenuRepository) UpsertMenus(menus []model.Menu) error {
	if len(menus) == 0 {
		return nil
	}

	// 트랜잭션 시작
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	// Defer rollback in case of panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // Re-panic after rollback
		}
	}()

	// 제약 조건 검사 지연 설정 (트랜잭션 내에서만 유효)
	// PostgreSQL 기준. 다른 DB는 구문이 다를 수 있음.
	if err := tx.Exec("SET CONSTRAINTS ALL DEFERRED").Error; err != nil {
		tx.Rollback() // 롤백 시도
		return fmt.Errorf("failed to set constraints deferred: %w", err)
	}

	// 모든 컬럼에 대해 충돌 시 업데이트 (ID 기준)
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"parent_id", "display_name", "res_type", "is_action", "priority", "menu_number"}),
	}).Create(&menus).Error; err != nil {
		tx.Rollback() // 롤백
		tx.Rollback() // 롤백
		return fmt.Errorf("failed to upsert menus in transaction: %w", err)
	}

	// --- Add logic to ensure resource type and create permissions ---
	// 1. Ensure 'menu' resource type exists
	resourceType := model.ResourceType{
		FrameworkID: "menu",
		ID:          "menu",
		Name:        "Menu Item",
		Description: "Represents a menu item resource",
	}
	// Use Clauses OnConflict to do nothing if the resource type already exists
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "framework_id"}, {Name: "id"}},
		DoNothing: true,
	}).Create(&resourceType).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to ensure menu resource type exists: %w", err)
	}

	// 2. Create/Update permissions for each menu item
	for _, menu := range menus {
		permissionID := fmt.Sprintf("menu:menu:view:%s", menu.ID)
		perm := model.MciamPermission{
			ID:             permissionID,
			FrameworkID:    "menu",
			ResourceTypeID: "menu",
			Action:         fmt.Sprintf("view:%s", menu.ID),               // Store the specific menu ID in action
			Name:           fmt.Sprintf("View Menu %s", menu.DisplayName), // Use DisplayName for Name
			Description:    fmt.Sprintf("Permission to view the '%s' menu item.", menu.DisplayName),
		}
		// Upsert the permission based on the unique ID
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "description", "updated_at"}), // Update name/desc if needed
		}).Create(&perm).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert permission for menu %s: %w", menu.ID, err)
		}
	}
	// --- End of added logic ---

	// 트랜잭션 커밋 (이 시점에 지연된 제약 조건 검사 발생)
	if err := tx.Commit().Error; err != nil {
		// Rollback might have already happened automatically on commit error
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindMappedMenusByRole returns menu IDs mapped to the given role
func (r *MenuRepository) FindMappedMenusByRole(roleID uint) ([]string, error) {
	var menuIDs []string
	err := r.db.Model(&model.RoleMenuMapping{}).
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs).Error
	return menuIDs, err
}

// FindAll 메뉴를 필터링하여 조회
func (r *MenuRepository) FindAll(req *model.MenuFilterRequest) ([]*model.Menu, error) {
	var menus []*model.Menu
	query := r.db.Model(&model.Menu{})

	if len(req.MenuNames) > 0 {
		query = query.Where("name IN ?", req.MenuNames)
	}
	if len(req.MenuIDs) > 0 {
		query = query.Where("id IN ?", req.MenuIDs)
	}

	// GORM은 기본적으로 UpdatedAt DESC 정렬을 시도할 수 있으므로 명시적 정렬 추가
	if err := query.Order("priority asc, menu_number asc").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

// CreateRoleMenuMappings 역할-메뉴 매핑을 생성합니다
func (r *MenuRepository) CreateRoleMenuMappings(mappings []*model.RoleMenuMapping) error {
	return r.db.Create(mappings).Error
}

// DeleteMapping 역할-메뉴 매핑 삭제
func (r *MenuRepository) DeleteRoleMenuMapping(mappings []*model.RoleMenuMapping) error {
	query := r.db.Delete(mappings)
	return query.Error
}

// 해당 role 과 매핑된 모든 메뉴 매핑 삭제
func (r *MenuRepository) DeleteRoleMenuMappingsByRoleID(roleID uint) error {
	query := r.db.Where("role_id = ?", roleID).Delete(&model.RoleMenuMapping{})
	return query.Error
}
