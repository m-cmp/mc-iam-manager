package service

import (
	"errors"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// OrganizationService 조직 비즈니스 로직
type OrganizationService struct {
	db      *gorm.DB
	orgRepo *repository.OrganizationRepository
}

// NewOrganizationService OrganizationService 생성자
func NewOrganizationService(db *gorm.DB) *OrganizationService {
	return &OrganizationService{
		db:      db,
		orgRepo: repository.NewOrganizationRepository(db),
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
				return nil, fmt.Errorf("parent organization not found: %d", *req.ParentID)
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

// buildOrganizationTree 평면 목록을 Tree 구조로 변환 (내부 함수)
func buildOrganizationTree(flat []model.OrganizationTree) []model.OrganizationTree {
	// ID → 인덱스 맵
	indexMap := make(map[uint]int)
	for i, org := range flat {
		indexMap[org.ID] = i
	}

	roots := []model.OrganizationTree{}
	nodes := make([]model.OrganizationTree, len(flat))
	copy(nodes, flat)

	for i := range nodes {
		org := &nodes[i]
		if org.ParentID == nil {
			roots = append(roots, *org)
		} else {
			parentIdx, ok := indexMap[*org.ParentID]
			if ok {
				nodes[parentIdx].Children = append(nodes[parentIdx].Children, *org)
			}
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

	// 부모 변경
	if req.ParentID != current.ParentID {
		if err := s.validateNoCircularReference(id, req.ParentID); err != nil {
			return err
		}

		var newParentCode string
		if req.ParentID != nil {
			parent, err := s.orgRepo.FindByID(*req.ParentID)
			if err != nil {
				return fmt.Errorf("parent organization not found: %d", *req.ParentID)
			}
			newParentCode = parent.OrganizationCode
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

		updates["parent_id"] = req.ParentID
		updates["organization_code"] = newCode
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
func (s *OrganizationService) DeleteOrganization(id uint) error {
	// 조직 존재 확인
	if _, err := s.orgRepo.FindByID(id); err != nil {
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

	return s.orgRepo.Delete(id)
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

// GetUserOrganizations 사용자가 소속된 조직 목록 조회
func (s *OrganizationService) GetUserOrganizations(userID uint) ([]model.Organization, error) {
	return s.orgRepo.FindUserOrganizations(userID)
}

// GetOrganizationUsers 조직에 소속된 사용자 목록 조회
func (s *OrganizationService) GetOrganizationUsers(orgID uint) ([]model.User, error) {
	return s.orgRepo.FindOrganizationUsers(orgID)
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
