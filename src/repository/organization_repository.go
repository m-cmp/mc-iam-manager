package repository

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrOrganizationNotFound      = errors.New("organization not found")
	ErrOrganizationNameDuplicate = errors.New("organization name already exists under the same parent")
	ErrOrganizationCodeDuplicate = errors.New("organization code already exists")
	ErrOrganizationHasChildren   = errors.New("organization has child organizations")
	ErrOrganizationHasUsers      = errors.New("organization has assigned users")
	ErrMaxOrganizationsPerLevel  = errors.New("maximum 99 organizations per level reached")
	ErrCircularReference         = errors.New("circular reference detected")
)

// OrganizationRepository 조직 데이터 관리
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository OrganizationRepository 생성자
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// --- CRUD ---

// Create 조직 생성
func (r *OrganizationRepository) Create(org *model.Organization) error {
	return r.db.Create(org).Error
}

// FindByID 조직 ID로 조회
func (r *OrganizationRepository) FindByID(id uint) (*model.Organization, error) {
	var org model.Organization
	if err := r.db.First(&org, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("error finding organization by id %d: %w", id, err)
	}
	return &org, nil
}

// FindByCode 조직 코드로 조회
func (r *OrganizationRepository) FindByCode(code string) (*model.Organization, error) {
	var org model.Organization
	if err := r.db.Where("organization_code = ?", code).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("error finding organization by code %s: %w", code, err)
	}
	return &org, nil
}

// FindAll 전체 조직 목록 조회 (평면)
func (r *OrganizationRepository) FindAll() ([]model.Organization, error) {
	var orgs []model.Organization
	if err := r.db.Order("organization_code ASC").Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("error finding all organizations: %w", err)
	}
	return orgs, nil
}

// FindChildren 직계 하위 조직 조회
func (r *OrganizationRepository) FindChildren(parentID uint) ([]model.Organization, error) {
	var orgs []model.Organization
	if err := r.db.Where("parent_id = ?", parentID).Order("organization_code ASC").Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("error finding children of organization %d: %w", parentID, err)
	}
	return orgs, nil
}

// Update 조직 정보 수정
func (r *OrganizationRepository) Update(id uint, updates map[string]interface{}) error {
	result := r.db.Model(&model.Organization{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationNotFound
	}
	return nil
}

// Delete 조직 삭제
func (r *OrganizationRepository) Delete(id uint) error {
	result := r.db.Delete(&model.Organization{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationNotFound
	}
	return nil
}

// UpsertOrganizations 조직 목록을 Upsert (organization_code 기준, 멱등성 보장)
// 부모-우선 순서로 전달되어야 함 (호출자 책임)
// ParentID는 organization_code에서 마지막 2자리 제거하여 내부적으로 조회
func (r *OrganizationRepository) UpsertOrganizations(orgs []model.Organization) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		codeToID := make(map[string]uint, len(orgs))

		for i := range orgs {
			// 부모 코드 유도: 코드 길이 > 2이면 마지막 2자리 제거
			code := orgs[i].OrganizationCode
			if len(code) > 2 {
				parentCode := code[:len(code)-2]
				if parentID, ok := codeToID[parentCode]; ok {
					orgs[i].ParentID = &parentID
				} else {
					// DB에서 부모 조회 (이미 존재하는 경우)
					var parent model.Organization
					if err := tx.Where("organization_code = ?", parentCode).First(&parent).Error; err == nil {
						orgs[i].ParentID = &parent.ID
					}
				}
			}

			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "organization_code"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "description"}),
			}).Create(&orgs[i]).Error
			if err != nil {
				return fmt.Errorf("error upserting organization code=%s: %w", code, err)
			}

			// 생성/업데이트 후 ID 조회 (OnConflict 시 Create가 ID를 채우지 않을 수 있음)
			var loaded model.Organization
			if err := tx.Where("organization_code = ?", code).First(&loaded).Error; err != nil {
				return fmt.Errorf("error reloading organization code=%s: %w", code, err)
			}
			codeToID[code] = loaded.ID
		}
		return nil
	})
}

// --- 조직 코드 자동 생성 ---

// GenerateOrganizationCode 계층적 2자리 단위 조직 코드 자동 생성
// parentCode가 빈 문자열이면 최상위 조직 코드 생성 (01, 02, ...)
// parentCode가 있으면 하위 조직 코드 생성 (parentCode + 01, 02, ...)
func (r *OrganizationRepository) GenerateOrganizationCode(parentCode string) (string, error) {
	prefix := parentCode
	targetLength := len(prefix) + 2

	// 동일 레벨에서 마지막 코드 조회
	var lastOrg model.Organization
	err := r.db.Where("organization_code LIKE ? AND LENGTH(organization_code) = ?",
		prefix+"%", targetLength).
		Order("organization_code DESC").
		First(&lastOrg).Error

	nextNumber := 1
	if err == nil {
		// 마지막 조직 코드에서 자신의 2자리 추출
		lastSuffix := lastOrg.OrganizationCode[len(prefix):]
		lastNum, parseErr := strconv.Atoi(lastSuffix)
		if parseErr == nil {
			nextNumber = lastNum + 1
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("error querying last organization code: %w", err)
	}

	if nextNumber > 99 {
		return "", ErrMaxOrganizationsPerLevel
	}

	return fmt.Sprintf("%s%02d", prefix, nextNumber), nil
}

// --- 검증 ---

// ExistsNameUnderParent 동일 부모 하 이름 중복 검사
func (r *OrganizationRepository) ExistsNameUnderParent(name string, parentID *uint, excludeID *uint) (bool, error) {
	query := r.db.Model(&model.Organization{}).Where("name = ?", name)

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsCode 조직 코드 중복 검사
func (r *OrganizationRepository) ExistsCode(code string, excludeID *uint) (bool, error) {
	query := r.db.Model(&model.Organization{}).Where("organization_code = ?", code)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasChildren 하위 조직 존재 여부 확인
func (r *OrganizationRepository) HasChildren(orgID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Organization{}).Where("parent_id = ?", orgID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasUsers 소속 사용자 존재 여부 확인
func (r *OrganizationRepository) HasUsers(orgID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.UserOrganization{}).Where("organization_id = ?", orgID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetDescendantIDs 하위 조직 ID 전체 조회 (재귀, 코드 재생성 시 사용)
// PostgreSQL CTE 사용
func (r *OrganizationRepository) GetDescendantIDs(orgID uint) ([]uint, error) {
	type Result struct {
		ID uint
	}
	var results []Result

	query := `
        WITH RECURSIVE descendants AS (
            SELECT id FROM mcmp_organizations WHERE parent_id = ?
            UNION ALL
            SELECT o.id FROM mcmp_organizations o
            INNER JOIN descendants d ON o.parent_id = d.id
        )
        SELECT id FROM descendants
    `
	if err := r.db.Raw(query, orgID).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("error getting descendant ids: %w", err)
	}

	ids := make([]uint, len(results))
	for i, res := range results {
		ids[i] = res.ID
	}
	return ids, nil
}

// --- Tree 조회 ---

// FindTreeFlat Tree 구조를 평면 목록으로 조회 (CTE, 레벨/경로 포함)
// 반환값은 OrganizationTree 슬라이스 (children 없음, level/path 포함)
func (r *OrganizationRepository) FindTreeFlat() ([]model.OrganizationTree, error) {
	type RawResult struct {
		ID               uint
		ParentID         *uint
		OrganizationCode string
		Name             string
		Description      string
		Level            int
		Path             string
		UserCount        int
	}
	var results []RawResult

	query := `
        WITH RECURSIVE org_tree AS (
            SELECT
                o.id,
                o.parent_id,
                o.organization_code,
                o.name,
                o.description,
                1 AS level,
                o.name::text AS path,
                (SELECT COUNT(*) FROM mcmp_user_organizations uo WHERE uo.organization_id = o.id) AS user_count
            FROM mcmp_organizations o
            WHERE o.parent_id IS NULL

            UNION ALL

            SELECT
                o.id,
                o.parent_id,
                o.organization_code,
                o.name,
                o.description,
                ot.level + 1,
                ot.path || '/' || o.name,
                (SELECT COUNT(*) FROM mcmp_user_organizations uo WHERE uo.organization_id = o.id) AS user_count
            FROM mcmp_organizations o
            INNER JOIN org_tree ot ON o.parent_id = ot.id
        )
        SELECT * FROM org_tree ORDER BY organization_code ASC
    `
	if err := r.db.Raw(query).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("error querying organization tree: %w", err)
	}

	trees := make([]model.OrganizationTree, len(results))
	for i, res := range results {
		trees[i] = model.OrganizationTree{
			ID:               res.ID,
			ParentID:         res.ParentID,
			OrganizationCode: res.OrganizationCode,
			Name:             res.Name,
			Description:      res.Description,
			Level:            res.Level,
			Path:             "/" + res.Path,
			UserCount:        res.UserCount,
		}
	}
	return trees, nil
}

// --- 사용자-조직 매핑 ---

// AssignUserToOrganizations 사용자를 조직에 할당 (단일 조직 기준)
func (r *OrganizationRepository) AssignUserToOrganizations(userID uint, orgIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, orgID := range orgIDs {
			mapping := model.UserOrganization{
				UserID:         userID,
				OrganizationID: orgID,
			}
			// ON CONFLICT DO NOTHING (이미 할당된 경우 무시)
			if err := tx.Where(mapping).FirstOrCreate(&mapping).Error; err != nil {
				return fmt.Errorf("error assigning user %d to organization %d: %w", userID, orgID, err)
			}
		}
		return nil
	})
}

// RemoveUserFromOrganization 사용자-조직 매핑 제거 (단건)
func (r *OrganizationRepository) RemoveUserFromOrganization(userID, orgID uint) error {
	result := r.db.Where("user_id = ? AND organization_id = ?", userID, orgID).
		Delete(&model.UserOrganization{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user is not assigned to this organization")
	}
	return nil
}

// FindUserOrganizations 사용자가 소속된 조직 목록 조회
func (r *OrganizationRepository) FindUserOrganizations(userID uint) ([]model.Organization, error) {
	var orgs []model.Organization
	if err := r.db.Joins("JOIN mcmp_user_organizations uo ON uo.organization_id = mcmp_organizations.id").
		Where("uo.user_id = ?", userID).
		Order("mcmp_organizations.organization_code ASC").
		Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("error finding organizations for user %d: %w", userID, err)
	}
	return orgs, nil
}

// FindOrganizationUsers 조직에 소속된 사용자 목록 조회
func (r *OrganizationRepository) FindOrganizationUsers(orgID uint) ([]model.User, error) {
	var users []model.User
	if err := r.db.Joins("JOIN mcmp_user_organizations uo ON uo.user_id = mcmp_users.id").
		Where("uo.organization_id = ?", orgID).
		Order("mcmp_users.username ASC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("error finding users for organization %d: %w", orgID, err)
	}
	return users, nil
}

// CountUserOrganizations 사용자 소속 조직 수 조회
func (r *OrganizationRepository) CountUserOrganizations(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.UserOrganization{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// UpdateDescendantCodes 하위 조직 코드 일괄 업데이트 (부모 이동 시)
// oldPrefix를 가진 모든 하위 조직의 코드를 newPrefix로 변경
func (r *OrganizationRepository) UpdateDescendantCodes(oldPrefix, newPrefix string) error {
	return r.db.Exec(
		`UPDATE mcmp_organizations
         SET organization_code = ? || SUBSTRING(organization_code FROM LENGTH(?)+1)
         WHERE organization_code LIKE ? AND organization_code != ?`,
		newPrefix, oldPrefix, oldPrefix+"%", oldPrefix,
	).Error
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
