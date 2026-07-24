package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// ErrMaxDepthExceeded 조직 이동 시 최대 깊이(10단계) 초과
var ErrMaxDepthExceeded = errors.New("organization tree depth would exceed maximum (10 levels)")

// OrganizationService 조직 비즈니스 로직
type OrganizationService struct {
	db        *gorm.DB
	orgRepo   *repository.OrganizationRepository
	kcService KeycloakService
}

// NewOrganizationService OrganizationService 생성자
func NewOrganizationService(db *gorm.DB) *OrganizationService {
	return &OrganizationService{
		db:        db,
		orgRepo:   repository.NewOrganizationRepository(db),
		kcService: NewKeycloakService(),
	}
}

// --- 조직 CRUD ---

// CreateOrganization 조직 생성
// 1. 부모 조직 존재 확인
// 2. 동일 부모 하 이름 중복 확인
// 3. 조직 코드 결정 (자동 생성 또는 직접 입력)
// 4. 조직 생성
func (s *OrganizationService) CreateOrganization(req *model.CreateOrganizationRequest) (*model.Organization, error) {
	// 1. 부모 조직 존재 확인
	var parentCode string
	if req.ParentID != nil {
		parent, err := s.orgRepo.FindByID(*req.ParentID)
		if err != nil {
			if errors.Is(err, repository.ErrOrganizationNotFound) {
				return nil, fmt.Errorf("parent organization not found: %d: %w", *req.ParentID, repository.ErrOrganizationNotFound)
			}
			return nil, err
		}
		parentCode = parent.OrganizationCode
	}

	// 2. 동일 부모 하 이름 중복 확인
	exists, err := s.orgRepo.ExistsNameUnderParent(req.Name, req.ParentID, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrOrganizationNameDuplicate
	}

	// 3. 조직 코드 결정
	var orgCode string
	if req.OrganizationCode != "" {
		// 직접 입력 - 중복 확인
		codeExists, err := s.orgRepo.ExistsCode(req.OrganizationCode, nil)
		if err != nil {
			return nil, err
		}
		if codeExists {
			return nil, repository.ErrOrganizationCodeDuplicate
		}
		orgCode = req.OrganizationCode
	} else {
		// 자동 생성
		orgCode, err = s.orgRepo.GenerateOrganizationCode(parentCode)
		if err != nil {
			return nil, err
		}
	}

	// 4. 조직 생성
	org := &model.Organization{
		ParentID:         req.ParentID,
		OrganizationCode: orgCode,
		Name:             req.Name,
		Description:      req.Description,
	}
	if err := s.orgRepo.Create(org); err != nil {
		return nil, fmt.Errorf("error creating organization: %w", err)
	}
	return org, nil
}

// GetOrganizationByID 조직 ID로 조회
func (s *OrganizationService) GetOrganizationByID(id uint) (*model.Organization, error) {
	return s.orgRepo.FindByID(id)
}

// GetOrganizationByCode 조직 코드로 조회
func (s *OrganizationService) GetOrganizationByCode(code string) (*model.Organization, error) {
	return s.orgRepo.FindByCode(code)
}

// GetOrganizations 조직 목록 조회 (Tree 또는 평면)
// tree=true: Tree 구조로 반환 (재귀 children)
// tree=false (기본): 평면 목록 (level, path 포함)
func (s *OrganizationService) GetOrganizations(tree bool) (interface{}, error) {
	flatList, err := s.orgRepo.FindTreeFlat()
	if err != nil {
		return nil, err
	}

	if !tree {
		return flatList, nil
	}

	// 평면 목록을 Tree 구조로 변환
	return buildOrganizationTree(flatList), nil
}

// GetOrganizationTree 전체 조직을 계층 트리 구조로 반환 (RQ-M2-UG-034-01)
// 최대 10단계 깊이를 지원하며 각 노드에 children 배열을 포함한다.
func (s *OrganizationService) GetOrganizationTree() ([]model.OrganizationTree, error) {
	flatList, err := s.orgRepo.FindTreeFlat()
	if err != nil {
		return nil, err
	}
	return buildOrganizationTree(flatList), nil
}

// GetOrganizationSubtree 특정 조직을 루트로 하는 하위 트리를 반환 (RQ-M2-UG-034-02)
func (s *OrganizationService) GetOrganizationSubtree(orgID uint) ([]model.OrganizationTree, error) {
	if _, err := s.orgRepo.FindByID(orgID); err != nil {
		return nil, err
	}

	flatList, err := s.orgRepo.FindSubtreeFlat(orgID)
	if err != nil {
		return nil, err
	}
	return buildOrganizationTree(flatList), nil
}

// MoveOrganization 조직을 트리 내 다른 위치로 이동 (RQ-M2-UG-035-01)
// 이동 시 하위 조직 코드 자동 재생성, 최대 10단계 깊이 초과 시 오류
func (s *OrganizationService) MoveOrganization(orgID uint, req *model.MoveOrganizationRequest) error {
	current, err := s.orgRepo.FindByID(orgID)
	if err != nil {
		return err
	}

	// 자기 자신 또는 하위 조직으로의 이동 방지
	if err := s.validateNoCircularReference(orgID, req.NewParentID); err != nil {
		return err
	}

	// 새 부모 코드 조회 (nil이면 최상위)
	newParentCode := ""
	if req.NewParentID != nil {
		newParent, err := s.orgRepo.FindByID(*req.NewParentID)
		if err != nil {
			return fmt.Errorf("new parent organization not found: %d: %w", *req.NewParentID, repository.ErrOrganizationNotFound)
		}
		newParentCode = newParent.OrganizationCode

		// 이동 후 최대 깊이(10단계) 초과 확인
		parentDepth, err := s.orgRepo.GetAncestorDepth(*req.NewParentID)
		if err != nil {
			return err
		}
		subtreeDepth, err := s.orgRepo.GetSubtreeDepth(orgID)
		if err != nil {
			return err
		}
		if parentDepth+subtreeDepth > 10 {
			return ErrMaxDepthExceeded
		}
	} else {
		// 최상위로 이동 시 서브트리 깊이 자체가 10을 넘으면 불가
		subtreeDepth, err := s.orgRepo.GetSubtreeDepth(orgID)
		if err != nil {
			return err
		}
		if subtreeDepth > 10 {
			return ErrMaxDepthExceeded
		}
	}

	// 새 코드 생성
	newCode, err := s.orgRepo.GenerateOrganizationCode(newParentCode)
	if err != nil {
		return err
	}

	// 하위 조직 코드 일괄 업데이트
	oldCode := current.OrganizationCode
	if err := s.orgRepo.UpdateDescendantCodes(oldCode, newCode); err != nil {
		return fmt.Errorf("error updating descendant codes: %w", err)
	}

	// 대상 조직 parent_id + code 업데이트
	updates := map[string]interface{}{
		"parent_id":         req.NewParentID,
		"organization_code": newCode,
	}
	return s.orgRepo.Update(orgID, updates)
}

// CheckOrganizationDeletable 조직 삭제 가능 여부 확인 (RQ-M2-UG-036-01)
func (s *OrganizationService) CheckOrganizationDeletable(orgID uint) (*model.OrganizationDeletableResponse, error) {
	if _, err := s.orgRepo.FindByID(orgID); err != nil {
		return nil, err
	}

	hasChildren, err := s.orgRepo.HasChildren(orgID)
	if err != nil {
		return nil, err
	}
	if hasChildren {
		return &model.OrganizationDeletableResponse{
			Deletable: false,
			Reason:    "하위 조직이 존재합니다. cascade=true 옵션을 사용하거나 하위 조직을 먼저 삭제해주세요.",
		}, nil
	}

	hasUsers, err := s.orgRepo.HasUsers(orgID)
	if err != nil {
		return nil, err
	}
	if hasUsers {
		return &model.OrganizationDeletableResponse{
			Deletable: false,
			Reason:    "소속 사용자가 있습니다. cascade=true 옵션을 사용하거나 사용자를 먼저 제거해주세요.",
		}, nil
	}

	return &model.OrganizationDeletableResponse{Deletable: true}, nil
}

// DeleteOrganizationCascade 조직 및 모든 하위 조직/사용자 매핑을 cascade 삭제 (RQ-M2-UG-036-02)
func (s *OrganizationService) DeleteOrganizationCascade(ctx context.Context, orgID uint) error {
	subtree, err := s.orgRepo.FindSubtreeOrganizations(orgID)
	if err != nil {
		return err
	}
	if len(subtree) == 0 {
		return repository.ErrOrganizationNotFound
	}

	if err := s.orgRepo.DeleteCascade(orgID); err != nil {
		return err
	}

	// Keycloak 그룹 정리 (DB는 이미 삭제됨, best-effort)
	var kcErrs []error
	for _, org := range subtree {
		if err := s.kcService.DeleteGroup(ctx, org.Name); err != nil {
			kcErrs = append(kcErrs, fmt.Errorf("group '%s': %w", org.Name, err))
		}
	}
	if len(kcErrs) > 0 {
		return fmt.Errorf("keycloak group cleanup failed for %d group(s) (DB already updated): %w", len(kcErrs), errors.Join(kcErrs...))
	}
	return nil
}

// SearchOrganizations name/code 필터로 조직 목록 검색 (평면 목록 반환)
func (s *OrganizationService) SearchOrganizations(name, code string) ([]model.Organization, error) {
	return s.orgRepo.FindByFilter(name, code)
}

// buildOrganizationTree 평면 목록을 Tree 구조로 변환 (내부 함수)
// flat은 organization_code ASC 정렬 상태 (01, 0101, 010101 ...)
// 깊은 노드부터 역순 처리: children이 먼저 채워진 후 부모에 복사되므로 전체 트리 유지
func buildOrganizationTree(flat []model.OrganizationTree) []model.OrganizationTree {
	nodes := make([]model.OrganizationTree, len(flat))
	for i := range flat {
		nodes[i] = flat[i]
		nodes[i].Children = nil
	}

	indexMap := make(map[uint]int, len(nodes))
	for i := range nodes {
		indexMap[nodes[i].ID] = i
	}

	// 역순(깊은 노드 먼저) 처리: 자식이 채워진 후 부모에 값 복사
	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].ParentID != nil {
			if parentIdx, ok := indexMap[*nodes[i].ParentID]; ok {
				nodes[parentIdx].Children = append(nodes[parentIdx].Children, nodes[i])
			}
		}
	}

	roots := make([]model.OrganizationTree, 0)
	for i := range nodes {
		if nodes[i].ParentID == nil {
			roots = append(roots, nodes[i])
		}
	}
	return roots
}

// UpdateOrganization 조직 정보 수정
// 부모 변경 시: 순환 참조 검증 + 하위 조직 코드 재생성
func (s *OrganizationService) UpdateOrganization(id uint, req *model.UpdateOrganizationRequest) error {
	// 현재 조직 조회
	current, err := s.orgRepo.FindByID(id)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{}

	// 이름 수정
	if req.Name != "" && req.Name != current.Name {
		exists, err := s.orgRepo.ExistsNameUnderParent(req.Name, req.ParentID, &id)
		if err != nil {
			return err
		}
		if exists {
			return repository.ErrOrganizationNameDuplicate
		}
		updates["name"] = req.Name
	}

	// 설명 수정
	if req.Description != current.Description {
		updates["description"] = req.Description
	}

	// 부모 변경 (req.ParentID가 nil이면 부모 변경 요청 없음으로 간주)
	if req.ParentID != nil {
		parentChanged := current.ParentID == nil || *req.ParentID != *current.ParentID
		if parentChanged {
			if err := s.validateNoCircularReference(id, req.ParentID); err != nil {
				return err
			}

			parent, err := s.orgRepo.FindByID(*req.ParentID)
			if err != nil {
				return fmt.Errorf("parent organization not found: %d", *req.ParentID)
			}

			// 새 코드 생성
			newCode, err := s.orgRepo.GenerateOrganizationCode(parent.OrganizationCode)
			if err != nil {
				return err
			}

			// 하위 조직 코드 일괄 업데이트
			oldCode := current.OrganizationCode
			if err := s.orgRepo.UpdateDescendantCodes(oldCode, newCode); err != nil {
				return fmt.Errorf("error updating descendant codes: %w", err)
			}

			updates["parent_id"] = req.ParentID
			updates["organization_code"] = newCode
		}
	}

	// 코드 직접 수정
	if req.OrganizationCode != "" && req.OrganizationCode != current.OrganizationCode {
		exists, err := s.orgRepo.ExistsCode(req.OrganizationCode, &id)
		if err != nil {
			return err
		}
		if exists {
			return repository.ErrOrganizationCodeDuplicate
		}
		updates["organization_code"] = req.OrganizationCode
	}

	if len(updates) == 0 {
		return nil // 변경 사항 없음
	}

	return s.orgRepo.Update(id, updates)
}

// DeleteOrganization 조직 삭제 (하위 조직/소속 사용자 존재 시 차단)
func (s *OrganizationService) DeleteOrganization(ctx context.Context, id uint) error {
	// 조직 존재 확인
	org, err := s.orgRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 하위 조직 확인
	hasChildren, err := s.orgRepo.HasChildren(id)
	if err != nil {
		return err
	}
	if hasChildren {
		return repository.ErrOrganizationHasChildren
	}

	// 소속 사용자 확인
	hasUsers, err := s.orgRepo.HasUsers(id)
	if err != nil {
		return err
	}
	if hasUsers {
		return repository.ErrOrganizationHasUsers
	}

	if err := s.orgRepo.Delete(id); err != nil {
		return err
	}

	// Keycloak 그룹 정리 (DB는 이미 삭제됨, best-effort)
	if err := s.kcService.DeleteGroup(ctx, org.Name); err != nil {
		return fmt.Errorf("keycloak group cleanup failed (DB already updated): %w", err)
	}
	return nil
}

// --- 사용자-조직 매핑 ---

// AssignUserToOrganizations 사용자를 조직에 할당 (다중)
func (s *OrganizationService) AssignUserToOrganizations(userID uint, orgIDs []uint) error {
	// 조직 존재 확인
	for _, orgID := range orgIDs {
		if _, err := s.orgRepo.FindByID(orgID); err != nil {
			return fmt.Errorf("organization not found: %d", orgID)
		}
	}
	return s.orgRepo.AssignUserToOrganizations(userID, orgIDs)
}

// RemoveUserFromOrganization 사용자-조직 매핑 제거
func (s *OrganizationService) RemoveUserFromOrganization(userID, orgID uint) error {
	return s.orgRepo.RemoveUserFromOrganization(userID, orgID)
}

// GetUserOrganizations 사용자가 소속된 조직 목록 조회 (계층 정보 포함)
func (s *OrganizationService) GetUserOrganizations(userID uint) ([]model.OrganizationTree, error) {
	return s.orgRepo.FindUserOrganizations(userID)
}

// GetUserOrganizationsWithHierarchy 사용자가 소속된 조직 목록을 계층 정보(path, level) 포함하여 조회
func (s *OrganizationService) GetUserOrganizationsWithHierarchy(userID uint) ([]model.OrganizationTree, error) {
	orgs, err := s.orgRepo.FindUserOrganizations(userID)
	if err != nil {
		return nil, err
	}

	// 전체 트리 평면 목록 조회 (path/level 계산용)
	flatAll, err := s.orgRepo.FindTreeFlat()
	if err != nil {
		return nil, err
	}

	// ID → tree 노드 맵 구성
	treeMap := make(map[uint]model.OrganizationTree, len(flatAll))
	for _, node := range flatAll {
		treeMap[node.ID] = node
	}

	// 사용자 소속 조직에 계층 정보 적용
	result := make([]model.OrganizationTree, 0, len(orgs))
	for _, org := range orgs {
		if node, ok := treeMap[org.ID]; ok {
			result = append(result, node)
		} else {
			// fallback: 기본 정보만
			result = append(result, model.OrganizationTree{
				ID:               org.ID,
				ParentID:         org.ParentID,
				OrganizationCode: org.OrganizationCode,
				Name:             org.Name,
				Description:      org.Description,
				CreatedAt:        org.CreatedAt,
				UpdatedAt:        org.UpdatedAt,
			})
		}
	}
	return result, nil
}

// ReplaceUserGroups 사용자의 그룹 멤버십을 전체 교체 (기존 제거 후 신규 할당)
func (s *OrganizationService) ReplaceUserGroups(userID uint, groupIDs []uint) error {
	// 신규 그룹 존재 확인
	for _, gID := range groupIDs {
		if _, err := s.orgRepo.FindByID(gID); err != nil {
			return fmt.Errorf("group not found: %d", gID)
		}
	}

	// 기존 그룹 조회
	currentOrgs, err := s.orgRepo.FindUserOrganizations(userID)
	if err != nil {
		return err
	}

	// 기존 그룹 제거
	for _, org := range currentOrgs {
		if err := s.orgRepo.RemoveUserFromOrganization(userID, org.ID); err != nil {
			// ErrUserOrganizationNotFound는 무시 (이미 제거됨)
			if !errors.Is(err, repository.ErrUserOrganizationNotFound) {
				return err
			}
		}
	}

	// 신규 그룹 할당
	if len(groupIDs) > 0 {
		return s.orgRepo.AssignUserToOrganizations(userID, groupIDs)
	}
	return nil
}

// GetOrganizationUsers 조직에 소속된 사용자 목록 조회
func (s *OrganizationService) GetOrganizationUsers(orgID uint) ([]model.User, error) {
	return s.orgRepo.FindOrganizationUsers(orgID)
}

// --- 조직 시드 ---

// LoadAndRegisterOrganizationsFromYAML YAML 파일에서 기본 조직 구조를 로드하여 DB에 Upsert
// filePath가 빈 문자열이면 기본 경로(asset/organization/organizations.yaml) 사용
// 파일이 없으면 WARN 로그 후 skip (soft failure)
func (s *OrganizationService) LoadAndRegisterOrganizationsFromYAML(filePath string) error {
	effectivePath := filePath
	if effectivePath == "" {
		assetPath := util.GetAssetPath()
		effectivePath = filepath.Join(assetPath, "organization", "organizations.yaml")
	}

	data, err := os.ReadFile(effectivePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[WARN] Organization seed file not found: %s, skipping", effectivePath)
			return nil
		}
		return fmt.Errorf("failed to read organization seed file %s: %w", effectivePath, err)
	}

	var seedData model.OrganizationSeedData
	if err := yaml.Unmarshal(data, &seedData); err != nil {
		return fmt.Errorf("failed to parse organization seed YAML: %w", err)
	}

	if len(seedData.Organizations) == 0 {
		log.Printf("[INFO] No organizations found in seed file %s, skipping", effectivePath)
		return nil
	}

	// 중첩 구조를 부모-우선(BFS) 슬라이스로 평탄화
	orgs := flattenOrganizationTree(seedData.Organizations)
	if err := s.orgRepo.UpsertOrganizations(orgs); err != nil {
		return fmt.Errorf("failed to upsert organizations: %w", err)
	}

	log.Printf("[INFO] Registered %d organizations from seed file", len(orgs))
	return nil
}

// flattenOrganizationTree 중첩 시드 구조를 부모-우선(BFS) 순서의 Organization 슬라이스로 변환
// ParentID는 UpsertOrganizations 내에서 organization_code로 자동 조회됨
func flattenOrganizationTree(items []model.OrganizationSeedItem) []model.Organization {
	var result []model.Organization
	queue := make([]model.OrganizationSeedItem, len(items))
	copy(queue, items)
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		result = append(result, model.Organization{
			OrganizationCode: item.OrganizationCode,
			Name:             item.Name,
			Description:      item.Description,
		})
		queue = append(queue, item.Children...)
	}
	return result
}

// --- 내부 유틸리티 ---

// validateNoCircularReference 순환 참조 검증
// orgID의 조상 체인에 newParentID가 포함되는지 확인
func (s *OrganizationService) validateNoCircularReference(orgID uint, newParentID *uint) error {
	if newParentID == nil {
		return nil // 최상위 조직으로 변경 - 순환 참조 불가
	}

	// 자기 자신을 부모로 설정 방지
	if orgID == *newParentID {
		return repository.ErrCircularReference
	}

	// 하위 조직을 부모로 설정 방지: newParentID의 조상 체인 탐색
	current := *newParentID
	visited := map[uint]bool{}

	for {
		if visited[current] {
			return repository.ErrCircularReference
		}
		visited[current] = true

		if current == orgID {
			return repository.ErrCircularReference
		}

		parent, err := s.orgRepo.FindByID(current)
		if err != nil {
			if errors.Is(err, repository.ErrOrganizationNotFound) {
				break
			}
			return err
		}

		if parent.ParentID == nil {
			break
		}

		current = *parent.ParentID
	}
	return nil
}
