package repository

import (
	"database/sql"

	"github.com/m-cmp/mc-iam-manager/model"
)

type MenuRepository struct {
	db *sql.DB
}

func NewMenuRepository(db *sql.DB) *MenuRepository {
	return &MenuRepository{
		db: db,
	}
}

// GetMenus 모든 메뉴 조회
func (r *MenuRepository) GetMenus() ([]model.Menu, error) {
	query := `
		SELECT id, parent_id, display_name, res_type, is_action, priority, menu_number 
		FROM menus 
		ORDER BY priority, menu_number`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []model.Menu
	for rows.Next() {
		var menu model.Menu
		var parentID sql.NullString

		err := rows.Scan(
			&menu.ID,
			&parentID,
			&menu.DisplayName,
			&menu.ResType,
			&menu.IsAction,
			&menu.Priority,
			&menu.MenuNumber,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			menu.ParentID = parentID.String
		}

		menus = append(menus, menu)
	}

	return menus, nil
}

// GetByID 메뉴 ID로 조회
func (r *MenuRepository) GetByID(id string) (*model.Menu, error) {
	query := `
		SELECT id, parent_id, display_name, res_type, is_action, priority, menu_number 
		FROM menus 
		WHERE id = $1`

	var menu model.Menu
	var parentID sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&menu.ID,
		&parentID,
		&menu.DisplayName,
		&menu.ResType,
		&menu.IsAction,
		&menu.Priority,
		&menu.MenuNumber,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		menu.ParentID = parentID.String
	}

	return &menu, nil
}

// Create 새 메뉴 생성
func (r *MenuRepository) Create(menu *model.Menu) error {
	query := `
		INSERT INTO menus (id, parent_id, display_name, res_type, is_action, priority, menu_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(query,
		menu.ID,
		menu.ParentID,
		menu.DisplayName,
		menu.ResType,
		menu.IsAction,
		menu.Priority,
		menu.MenuNumber,
	)
	return err
}

// Update 메뉴 정보 업데이트
func (r *MenuRepository) Update(menu *model.Menu) error {
	query := `
		UPDATE menus 
		SET parent_id = $2, display_name = $3, res_type = $4, is_action = $5, priority = $6, menu_number = $7
		WHERE id = $1`

	_, err := r.db.Exec(query,
		menu.ID,
		menu.ParentID,
		menu.DisplayName,
		menu.ResType,
		menu.IsAction,
		menu.Priority,
		menu.MenuNumber,
	)
	return err
}

// Delete 메뉴 삭제
func (r *MenuRepository) Delete(id string) error {
	query := `DELETE FROM menus WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
