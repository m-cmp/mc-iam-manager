package service

import (
	"context" // Added
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort" // Added
	"strconv"
	"strings"
	"time"

	"encoding/csv"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm" // Import gorm
)

const (
	defaultMenuViewType         = "local"
	defaultMenuFrameworkService = "mc-web-console-front"
	maxMenuPathLength           = 500
)

var (
	ErrInvalidViewType            = errors.New("invalid viewType")
	ErrFrameworkServiceRequired   = errors.New("frameworkService required")
	ErrPathTooLong                = errors.New("path too long")
)

func normalizeMenuResource(viewType, frameworkService, path string) (string, string, string) {
	if viewType == "" {
		viewType = defaultMenuViewType
	}
	if frameworkService == "" {
		frameworkService = defaultMenuFrameworkService
	}
	return viewType, frameworkService, path
}

func validateMenuResource(viewType, frameworkService, path string) error {
	switch viewType {
	case "local", "iframe", "popup":
	default:
		return ErrInvalidViewType
	}
	if (viewType == "iframe" || viewType == "popup") && frameworkService == "" {
		return ErrFrameworkServiceRequired
	}
	if len(path) > maxMenuPathLength {
		return ErrPathTooLong
	}
	return nil
}

func normalizeAndValidateMenuResource(viewType, frameworkService, path string) (string, string, string, error) {
	viewType, frameworkService, path = normalizeMenuResource(viewType, frameworkService, path)
	if err := validateMenuResource(viewType, frameworkService, path); err != nil {
		return "", "", "", err
	}
	return viewType, frameworkService, path, nil
}

func applyMenuResourceDefaults(menu *model.Menu) error {
	viewType, frameworkService, path, err := normalizeAndValidateMenuResource(
		menu.ViewType, menu.FrameworkService, menu.Path,
	)
	if err != nil {
		return err
	}
	menu.ViewType = viewType
	menu.FrameworkService = frameworkService
	menu.Path = path
	return nil
}

// MenuService 메뉴 관련 비즈니스 로직
type MenuService struct {
	db              *gorm.DB // Add db field
	menuRepo        *repository.MenuRepository
	userRepo        *repository.UserRepository            // Added dependency
	permissionRepo  *repository.MciamPermissionRepository // Use renamed repository type
	menuMappingRepo *repository.MenuMappingRepository
	roleRepo        *repository.RoleRepository
}

// NewMenuService 새 MenuService 인스턴스 생성
func NewMenuService(db *gorm.DB) *MenuService {
	return &MenuService{
		db:              db, // Store db
		menuRepo:        repository.NewMenuRepository(db),
		userRepo:        repository.NewUserRepository(db),
		permissionRepo:  repository.NewMciamPermissionRepository(db),
		menuMappingRepo: repository.NewMenuMappingRepository(db),
		roleRepo:        repository.NewRoleRepository(db),
	}
}

// GetAllMenusTree 모든 메뉴를 트리 구조로 조회 (관리자용)
func (s *MenuService) ListAllMenus(req *model.MenuFilterRequest) ([]*model.Menu, error) {

	allMenus, err := s.menuRepo.GetMenus(req) // Get all menus
	if err != nil {
		return nil, fmt.Errorf("failed to get all menus: %w", err)
	}

	return allMenus, nil
}

func (s *MenuService) GetAllMenusTree(req *model.MenuFilterRequest) ([]*model.MenuTreeNode, error) {
	allMenus, err := s.menuRepo.GetMenus(req) // Get all menus
	if err != nil {
		return nil, fmt.Errorf("failed to get all menus: %w", err)
	}
	if len(allMenus) == 0 {
		return []*model.MenuTreeNode{}, nil
	}

	// Use the existing helper function to build the tree
	tree := buildMenuTree(allMenus)
	return tree, nil
}

// BuildUserMenuTree 사용자의 플랫폼 역할에 따른 메뉴 트리 구성
func (s *MenuService) BuildUserMenuTree(ctx context.Context, platformRoleIDs []uint) ([]*model.MenuTreeNode, error) {
	req := &model.MenuMappingFilterRequest{}
	var allMenus []*model.Menu

	// 1. 각 플랫폼 역할에 매핑된 메뉴 ID들을 조회
	menuIDs, err := s.menuMappingRepo.FindMappedMenuIDs(req)
	if err != nil {
		return nil, err
	}

	// 2. 매핑된 메뉴 ID들의 상위 메뉴 ID들을 수집
	//parentIDs := []*string{}
	menuFilterRequest := &model.MenuFilterRequest{
		MenuIDs: menuIDs,
	}
	menus, err := s.menuRepo.GetMenus(menuFilterRequest)
	if err != nil {
		return nil, err
	}
	allMenus = append(allMenus, menus...)

	parentIDs, err := s.menuRepo.FindParentIDs(menuIDs)
	if err != nil {
		return nil, err
	}

	// for _, menuID := range menuIDs {
	// 	menu, err := s.menuRepo.FindMenuByID(menuID)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	// 상위 메뉴 ID가 있으면 수집
	// 	if menu.ParentID != "" {
	// 		parentIDs = append(parentIDs, &menu.ParentID)
	// 	}
	// 	allMenus = append(allMenus, menu)
	// }

	// 3. 모든 필요한 메뉴 ID를 하나의 맵으로 합침

	// 4. 수집된 메뉴 ID들로 메뉴 정보 조회
	parentMenuFilterRequest := &model.MenuFilterRequest{
		MenuIDs: parentIDs,
	}
	parentMenus, err := s.menuRepo.GetMenus(parentMenuFilterRequest)
	if err != nil {
		return nil, err
	}
	allMenus = append(allMenus, parentMenus...)

	// for _, platformMenuID := range parentIDs {
	// 	menu, err := s.menuRepo.FindMenuByID(platformMenuID)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	allMenus = append(allMenus, menu)
	// }

	// 5. 메뉴 트리 구성
	menuTree := buildMenuTree(allMenus)

	// 6. 정렬
	sortMenuTree(menuTree)

	return menuTree, nil
}

// Role에 따른 메뉴 목록록
func (s *MenuService) MenuList(req *model.MenuMappingFilterRequest) ([]*model.Menu, error) {
	// 1. 각 플랫폼 역할에 매핑된 메뉴 ID들을 조회
	menuIDMap := make(map[string]bool)
	menuIDs, err := s.menuMappingRepo.FindMappedMenuIDs(req)
	if err != nil {
		return nil, err
	}
	for _, id := range menuIDs {
		menuIDMap[*id] = true
	}

	// 2. 매핑된 메뉴 ID들의 상위 메뉴 ID들을 수집
	parentIDMap := make(map[string]bool)
	parentIDs := []*string{}
	menuFilterRequest := &model.MenuFilterRequest{
		MenuIDs: menuIDs,
	}
	menus, err := s.menuRepo.GetMenus(menuFilterRequest)
	if err != nil {
		return nil, err
	}

	for _, menu := range menus {
		parentIDMap[menu.ParentID] = true
		parentIDs = append(parentIDs, &menu.ParentID)
	}

	parentMenuFilterRequest := &model.MenuFilterRequest{
		MenuIDs: parentIDs,
	}
	parentMenus, err := s.menuRepo.GetMenus(parentMenuFilterRequest)
	if err != nil {
		return nil, err
	}

	// 중복 제거를 위한 map 사용
	uniqueMenus := make(map[string]*model.Menu)

	// 기존 메뉴 추가
	for _, menu := range menus {
		uniqueMenus[menu.ID] = menu
	}

	// 부모 메뉴 추가 (중복되는 경우 덮어쓰기)
	for _, menu := range parentMenus {
		uniqueMenus[menu.ID] = menu
	}

	// map에서 슬라이스로 변환
	result := make([]*model.Menu, 0, len(uniqueMenus))
	for _, menu := range uniqueMenus {
		result = append(result, menu)
	}

	return result, nil
}

// sortMenuTree 메뉴 트리를 정렬하는 헬퍼 함수
func sortMenuTree(nodes []*model.MenuTreeNode) {
	// 우선순위와 메뉴 번호로 정렬
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Priority != nodes[j].Priority {
			return nodes[i].Priority < nodes[j].Priority
		}
		return nodes[i].MenuNumber < nodes[j].MenuNumber
	})

	// 자식 노드들도 재귀적으로 정렬
	for _, node := range nodes {
		if len(node.Children) > 0 {
			sortMenuTree(node.Children)
		}
	}
}

// buildMenuTree 재귀적으로 메뉴 트리 구조 생성 (헬퍼 함수)
func buildMenuTree(menus []*model.Menu) []*model.MenuTreeNode {
	nodeMap := make(map[string]*model.MenuTreeNode, len(menus))
	var rootNodes []*model.MenuTreeNode

	// Create all nodes and put them in a map
	for i := range menus {
		node := &model.MenuTreeNode{
			Menu: *menus[i], // Copy menu data
		}
		nodeMap[node.ID] = node
	}

	// Build the tree structure
	for _, node := range nodeMap {
		if node.ParentID == "" {
			rootNodes = append(rootNodes, node)
		} else {
			parentNode, exists := nodeMap[node.ParentID]
			if exists {
				parentNode.Children = append(parentNode.Children, node)
			} else {
				// Orphan node (parent not accessible or doesn't exist), treat as root? Or log warning?
				// fmt.Printf("Warning: Parent menu %s not found or not accessible for menu %s\n", node.ParentID, node.ID)
				rootNodes = append(rootNodes, node) // Add orphans as root nodes for now
			}
		}
	}

	// Sort nodes at each level by priority, then menu_number
	sortNodes(rootNodes)

	return rootNodes
}

// sortNodes 재귀적으로 노드와 자식 노드 정렬 (헬퍼 함수)
func sortNodes(nodes []*model.MenuTreeNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Priority != nodes[j].Priority {
			return nodes[i].Priority < nodes[j].Priority
		}
		return nodes[i].MenuNumber < nodes[j].MenuNumber
	})
	for _, node := range nodes {
		if len(node.Children) > 0 {
			sortNodes(node.Children)
		}
	}
}

// GetMenus 모든 메뉴 조회 (Deprecated or internal use only)
func (s *MenuService) GetMenus(req *model.MenuFilterRequest) ([]*model.Menu, error) {
	// return s.menuRepo.GetMenus() // Keep original GetMenus for now
	var menus []*model.Menu
	menus, err := s.menuRepo.GetMenus(req)
	return menus, err
}

// GetByID 메뉴 ID로 조회
func (s *MenuService) GetMenuByID(id *string) (*model.Menu, error) {
	return s.menuRepo.FindMenuByID(id)
}

// Create 새 메뉴 생성
func (s *MenuService) Create(req *model.CreateMenuRequest) error {
	viewType, frameworkService, path, err := normalizeAndValidateMenuResource(
		req.ViewType, req.FrameworkService, req.Path,
	)
	if err != nil {
		return err
	}
	req.ViewType = viewType
	req.FrameworkService = frameworkService
	req.Path = path
	return s.menuRepo.CreateMenu(req)
}

// CreateWithRoleMappings 메뉴 생성 + 역할 매핑 (platform_admin 자동 포함)
func (s *MenuService) CreateWithRoleMappings(req *model.CreateMenuRequest) (*model.CreateMenuResponse, error) {
	// 1. admin(platform_admin) 역할 조회
	adminRole, err := s.roleRepo.FindRoleByRoleName("admin", constants.RoleTypePlatform)
	if err != nil {
		return nil, fmt.Errorf("admin 역할 조회 실패: %w", err)
	}
	if adminRole == nil {
		return nil, fmt.Errorf("admin 역할을 찾을 수 없습니다")
	}

	// 2. 매핑할 역할 ID 목록 구성 (platform_admin 자동 포함, 중복 제거)
	roleIDSet := map[uint]struct{}{adminRole.ID: {}}
	for _, rid := range req.RoleIDs {
		roleIDSet[rid] = struct{}{}
	}

	// 3. 요청 roleID 유효성 확인 (admin 제외)
	for rid := range roleIDSet {
		if rid == adminRole.ID {
			continue
		}
		role, err := s.roleRepo.FindRoleByRoleID(rid, constants.IAMRoleType(""))
		if err != nil {
			return nil, fmt.Errorf("역할 조회 실패 (roleId=%d): %w", rid, err)
		}
		if role == nil {
			return nil, fmt.Errorf("존재하지 않는 역할 ID입니다: %d", rid)
		}
	}

	// 4. 중복 제거된 역할 ID 슬라이스 생성
	finalRoleIDs := make([]uint, 0, len(roleIDSet))
	for rid := range roleIDSet {
		finalRoleIDs = append(finalRoleIDs, rid)
	}

	// 5. Menu 객체 생성
	priorityInt, err := util.StringToUint(req.Priority)
	if err != nil {
		return nil, fmt.Errorf("잘못된 priority 값: %w", err)
	}
	menuNumberInt, err := util.StringToUint(req.MenuNumber)
	if err != nil {
		return nil, fmt.Errorf("잘못된 menuNumber 값: %w", err)
	}
	isAction := false
	if req.IsAction != nil {
		isAction = *req.IsAction
	}
	menu := &model.Menu{
		ID:          req.ID,
		ParentID:    req.ParentID,
		DisplayName: req.DisplayName,
		ResType:     req.ResType,
		IsAction:    isAction,
		Priority:    priorityInt,
		MenuNumber:  menuNumberInt,
		ViewType:         req.ViewType,
		FrameworkService: req.FrameworkService,
		Path:             req.Path,
	}
	if err := applyMenuResourceDefaults(menu); err != nil {
		return nil, err
	}

	// 6. 트랜잭션: 메뉴 생성 + 역할 매핑
	mappings, err := s.menuRepo.CreateMenuWithRoleMappings(menu, finalRoleIDs)
	if err != nil {
		return nil, fmt.Errorf("메뉴 생성 및 역할 매핑 실패: %w", err)
	}

	return &model.CreateMenuResponse{
		Menu:         menu,
		RoleMappings: mappings,
	}, nil
}

// Update 메뉴 정보 부분 업데이트
func (s *MenuService) Update(id string, updates map[string]interface{}) error {
	_, hasViewType := updates["view_type"]
	_, hasFramework := updates["framework_service"]
	_, hasPath := updates["path"]
	if hasViewType || hasFramework || hasPath {
		existing, err := s.menuRepo.FindMenuByID(&id)
		if err != nil {
			return err
		}
		if existing == nil {
			return repository.ErrMenuNotFound
		}
		viewType := existing.ViewType
		frameworkService := existing.FrameworkService
		path := existing.Path
		if v, ok := updates["view_type"].(string); ok {
			viewType = v
		}
		if v, ok := updates["framework_service"].(string); ok {
			frameworkService = v
		}
		if v, ok := updates["path"].(string); ok {
			path = v
		}
		viewType, frameworkService, path, err = normalizeAndValidateMenuResource(
			viewType, frameworkService, path,
		)
		if err != nil {
			return err
		}
		updates["view_type"] = viewType
		updates["framework_service"] = frameworkService
		updates["path"] = path
	}

	return s.menuRepo.UpdateMenu(id, updates)
}

// Delete 메뉴 삭제
func (s *MenuService) Delete(id string) error {
	return s.menuRepo.DeleteMenuWithChildren(id)
}

// LoadAndRegisterMenusFromYAML YAML 파일에서 메뉴를 로드하여 DB에 등록(Upsert)
// filePath 쿼리 파라미터가 없으면 .env의 MC_WEB_CONSOLE_MENUYAML URL에서 다운로드 시도
func (s *MenuService) LoadAndRegisterMenusFromYAML(filePath string) error {
	effectiveFilePath := filePath
	downloaded := false

	// If filePath is not provided via query param
	if effectiveFilePath == "" {
		// Load .env file to get the URL (assuming .env is at project root)
		// .env path should be relative to project root when running the binary
		util.LoadEnvFiles()
		menuURL := os.Getenv("MC_WEB_CONSOLE_MENUYAML")

		// Default local path relative to project root
		assetPath := util.GetAssetPath()
		defaultLocalPath := filepath.Join(assetPath, "menu", "menu.yaml")

		if menuURL != "" && (strings.HasPrefix(menuURL, "http://") || strings.HasPrefix(menuURL, "https://")) {
			// Attempt to download from URL
			fmt.Printf("Attempting to download menu YAML from URL: %s\n", menuURL)
			resp, err := http.Get(menuURL)
			if err != nil {
				fmt.Printf("Warning: Failed to download menu YAML from %s: %v. Falling back to local path: %s\n", menuURL, err, defaultLocalPath)
				effectiveFilePath = defaultLocalPath
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					fmt.Printf("Warning: Failed to download menu YAML from %s (Status: %s). Falling back to local path: %s\n", menuURL, resp.Status, defaultLocalPath)
					effectiveFilePath = defaultLocalPath
				} else {
					bodyBytes, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Printf("Warning: Failed to read response body from %s: %v. Falling back to local path: %s\n", menuURL, err, defaultLocalPath)
						effectiveFilePath = defaultLocalPath
					} else {
						// Ensure target directory exists
						targetDir := filepath.Dir(defaultLocalPath)
						if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
							fmt.Printf("Warning: Failed to create directory %s: %v. Cannot save downloaded file.\n", targetDir, err)
							// Decide whether to proceed with potentially stale local file or error out
							effectiveFilePath = defaultLocalPath // Fallback to local path anyway
						} else {
							// Save downloaded content to the default local path
							if err := os.WriteFile(defaultLocalPath, bodyBytes, 0644); err != nil {
								fmt.Printf("Warning: Failed to write downloaded menu YAML to %s: %v. Using potentially stale local file.\n", defaultLocalPath, err)
								effectiveFilePath = defaultLocalPath // Fallback to local path
							} else {
								fmt.Printf("Successfully downloaded and saved menu YAML to %s\n", defaultLocalPath)
								effectiveFilePath = defaultLocalPath
								downloaded = true
							}
						}
					}
				}
			}
		} else if menuURL != "" {
			// If MC_WEB_CONSOLE_MENUYAML is set but not a URL, assume it's a local path relative to project root
			fmt.Printf("Using local menu YAML path from MC_WEB_CONSOLE_MENUYAML: %s\n", menuURL)
			// Assuming menuURL is relative to project root:
			effectiveFilePath = menuURL // Use the path directly
		} else {
			// If MC_WEB_CONSOLE_MENUYAML is not set, use the default local path
			fmt.Printf("MC_WEB_CONSOLE_MENUYAML not set or invalid URL, using default local path: %s\n", defaultLocalPath)
			effectiveFilePath = defaultLocalPath
		}
	}

	// 1. YAML 파일 로드 (from effectiveFilePath)
	menus, err := s.menuRepo.LoadMenusFromYAML(effectiveFilePath)
	if err != nil {
		// If download failed and local file also fails, return error
		return fmt.Errorf("failed to load menus from YAML file %s (downloaded: %v): %w", effectiveFilePath, downloaded, err)
	}

	if len(menus) == 0 {
		fmt.Printf("No menus found in YAML file %s, skipping registration.\n", effectiveFilePath)
		return nil // 처리할 메뉴 없음
	}

	// 2. home 메뉴가 있는지 확인하고 업데이트
	for _, menu := range menus {
		if menu.ID == "home" {
			// home 메뉴가 있으면 업데이트
			homeMenu := menu
			if err := applyMenuResourceDefaults(&homeMenu); err != nil {
				return fmt.Errorf("invalid home menu resource: %w", err)
			}
			if err := s.menuRepo.UpdateMenu("home", map[string]interface{}{
				"display_name":      homeMenu.DisplayName,
				"res_type":          homeMenu.ResType,
				"is_action":         homeMenu.IsAction,
				"priority":          homeMenu.Priority,
				"menu_number":       homeMenu.MenuNumber,
				"view_type":         homeMenu.ViewType,
				"framework_service": homeMenu.FrameworkService,
				"path":              homeMenu.Path,
			}); err != nil {
				fmt.Printf("Warning: Failed to update home menu: %v\n", err)
			}
			break
		}
	}

	// 3. 메뉴를 부모-자식 관계에 따라 정렬
	sort.Slice(menus, func(i, j int) bool {
		// 부모 ID가 없는 메뉴를 먼저
		if menus[i].ParentID == "" && menus[j].ParentID != "" {
			return true
		}
		if menus[i].ParentID != "" && menus[j].ParentID == "" {
			return false
		}
		// 둘 다 부모가 있거나 둘 다 부모가 없는 경우 ID로 정렬
		return menus[i].ID < menus[j].ID
	})

	// 4. DB에 Upsert (home 메뉴 제외)
	var menusToUpsert []model.Menu
	for _, menu := range menus {
		if menu.ID == "home" {
			continue
		}
		menuCopy := menu
		if err := applyMenuResourceDefaults(&menuCopy); err != nil {
			return fmt.Errorf("invalid menu resource for %s: %w", menu.ID, err)
		}
		menusToUpsert = append(menusToUpsert, menuCopy)
	}

	if len(menusToUpsert) > 0 {
		if err := s.menuRepo.UpsertMenus(menusToUpsert); err != nil {
			return fmt.Errorf("failed to upsert menus to database: %w", err)
		}
	}

	return nil
}

// RegisterMenusFromContent YAML 콘텐츠([]byte)를 파싱하여 DB에 등록(Upsert)
func (s *MenuService) RegisterMenusFromContent(yamlContent []byte) error {
	// 1. YAML 파싱
	var menuData struct { // 임시 구조체 사용
		Menus []model.Menu `yaml:"menus"`
	}
	// Use yaml.Unmarshal from the imported package
	err := yaml.Unmarshal(yamlContent, &menuData)
	if err != nil {
		return fmt.Errorf("error unmarshalling menu yaml content: %w", err)
	}

	menus := menuData.Menus
	if len(menus) == 0 {
		// log.Printf("No menus found in YAML content, skipping registration.")
		return nil // 처리할 메뉴 없음
	}

	for i := range menus {
		if err := applyMenuResourceDefaults(&menus[i]); err != nil {
			return fmt.Errorf("invalid menu resource for %s: %w", menus[i].ID, err)
		}
	}

	// 2. DB에 Upsert
	if err := s.menuRepo.UpsertMenus(menus); err != nil {
		return fmt.Errorf("failed to upsert menus to database: %w", err)
	}

	// log.Printf("Successfully registered %d menus from YAML content", len(menus))
	return nil
}

// ListMappedMenusByRole 플랫폼 역할에 매핑된 메뉴 목록 조회
func (s *MenuService) ListMappedMenusByRole(req *model.MenuMappingFilterRequest) ([]*model.Menu, error) {
	var menus []*model.Menu
	for _, roleID := range req.RoleIDs {
		roleIDInt, err := util.StringToUint(roleID)
		if err != nil {
			return nil, err
		}
		roleMenus, err := s.menuMappingRepo.FindMappedMenus(roleIDInt)
		if err != nil {
			return nil, err
		}
		menus = append(menus, roleMenus...)
	}

	return menus, nil
}

// rolePermissionEntry permission.yaml 역할 단위 권한 정의
// permissions → role → menus | operations | csps
type rolePermissionEntry struct {
	Role       string   `yaml:"role"`
	Menus      []string `yaml:"menus"`
	Operations []string `yaml:"operations"`
	Csps       []string `yaml:"csps"`
}

type rolePermissionFile struct {
	Permissions []rolePermissionEntry `yaml:"permissions"`
}

// InitializeMenuPermissionsFromCSV CSV 매트릭스로 역할-메뉴 권한을 시드합니다.
//
// Deprecated: permission.csv 시드는 제거 예정입니다.
// 신규/운영 시드는 InitializeMenuPermissionsFromYAML(asset/menu/permission.yaml)을 사용하세요.
func (s *MenuService) InitializeMenuPermissionsFromCSV(filePath string) error {
	effectiveFilePath, cleanup, err := s.resolvePermissionSeedPath(
		filePath, "permission.csv", ".csv",
	)
	if err != nil {
		return err
	}
	if cleanup != "" {
		defer os.Remove(cleanup)
	}
	return s.initializeMenuPermissionsFromCSVFile(effectiveFilePath)
}

// InitializeMenuPermissionsFromYAML 역할 중심 permission.yaml으로 권한을 시드합니다.
// filePath가 비어 있으면 확장자가 맞는 MC_WEB_CONSOLE_MENU_PERMISSIONS,
// 또는 asset/menu/permission.yaml을 사용합니다.
// 스키마: permissions → role → menus | operations | csps
func (s *MenuService) InitializeMenuPermissionsFromYAML(filePath string) error {
	effectiveFilePath, cleanup, err := s.resolvePermissionSeedPath(
		filePath, "permission.yaml", ".yaml",
	)
	if err != nil {
		return err
	}
	if cleanup != "" {
		defer os.Remove(cleanup)
	}
	return s.loadAndApplyMenuPermissionsFromYAML(effectiveFilePath)
}

// permissionSeedSourceMatchesExt는 source 경로/URL 확장자가 defaultExt와 맞는지 확인합니다.
// YAML(.yaml/.yml)은 서로 호환으로 취급하며, 쿼리스트링은 확장자 검사 전에 제거합니다.
func permissionSeedSourceMatchesExt(source, defaultExt string) bool {
	pathPart := strings.SplitN(source, "?", 2)[0]
	ext := strings.ToLower(filepath.Ext(pathPart))
	want := strings.ToLower(defaultExt)
	if want == ".yaml" || want == ".yml" {
		return ext == ".yaml" || ext == ".yml"
	}
	return ext == want
}

// resolvePermissionSeedPath 시드 파일 경로를 결정합니다.
// 우선순위: query filePath → 확장자가 맞는 MC_WEB_CONSOLE_MENU_PERMISSIONS →
// asset/menu/{defaultFileName}
// 공유 env의 확장자가 기대 포맷과 다르면 다운로드하지 않고 로컬 기본 파일로 fallback합니다.
func (s *MenuService) resolvePermissionSeedPath(
	filePath, defaultFileName, defaultExt string,
) (string, string, error) {
	if filePath != "" {
		return filePath, "", nil
	}

	util.LoadEnvFiles()
	assetPath := util.GetAssetPath()
	defaultLocalPath := filepath.Join(assetPath, "menu", defaultFileName)

	permissionSource := ""
	shared := strings.TrimSpace(os.Getenv("MC_WEB_CONSOLE_MENU_PERMISSIONS"))
	if shared != "" {
		if permissionSeedSourceMatchesExt(shared, defaultExt) {
			permissionSource = shared
		} else {
			fmt.Printf(
				"Warning: MC_WEB_CONSOLE_MENU_PERMISSIONS (%s) is not a %s source; "+
					"falling back to local %s.\n",
				shared, defaultExt, defaultLocalPath,
			)
		}
	}

	if permissionSource == "" {
		return defaultLocalPath, "", nil
	}

	if strings.HasPrefix(permissionSource, "http://") ||
		strings.HasPrefix(permissionSource, "https://") {
		fmt.Printf("Attempting to download permission seed from URL: %s\n", permissionSource)
		resp, err := http.Get(permissionSource)
		if err != nil {
			fmt.Printf(
				"Warning: Failed to download permission seed from %s: %v. Falling back to local file.\n",
				permissionSource, err,
			)
			return defaultLocalPath, "", nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Printf(
				"Warning: Failed to download permission seed from %s (Status: %s). Falling back to local file.\n",
				permissionSource, resp.Status,
			)
			return defaultLocalPath, "", nil
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf(
				"Warning: Failed to read permission seed from %s: %v. Falling back to local file.\n",
				permissionSource, err,
			)
			return defaultLocalPath, "", nil
		}
		ext := filepath.Ext(strings.SplitN(permissionSource, "?", 2)[0])
		if ext == "" {
			ext = defaultExt
		}
		tempFile, err := os.CreateTemp("", "permission-*"+ext)
		if err != nil {
			fmt.Printf(
				"Warning: Failed to create temp file: %v. Falling back to local file.\n", err,
			)
			return defaultLocalPath, "", nil
		}
		if _, err := tempFile.Write(bodyBytes); err != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			fmt.Printf(
				"Warning: Failed to write temp permission file: %v. Falling back to local file.\n",
				err,
			)
			return defaultLocalPath, "", nil
		}
		tempFile.Close()
		return tempFile.Name(), tempFile.Name(), nil
	}

	return permissionSource, "", nil
}

// loadAndApplyMenuPermissionsFromYAML 역할 중심 permission.yaml을 적용합니다.
func (s *MenuService) loadAndApplyMenuPermissionsFromYAML(filePath string) error {
	body, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read permission YAML: %w", err)
	}

	var data rolePermissionFile
	if err := yaml.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to parse permission YAML: %w", err)
	}
	if len(data.Permissions) == 0 {
		return fmt.Errorf("no permissions entries found in %s", filePath)
	}

	roleMenus := make(map[string][]string)
	for _, entry := range data.Permissions {
		roleName := strings.TrimSpace(entry.Role)
		if roleName == "" {
			return fmt.Errorf("permission entry missing role in %s", filePath)
		}
		if len(entry.Operations) > 0 {
			fmt.Printf(
				"Note: role %s operations(%d) reserved — menu seed only in this pass\n",
				roleName, len(entry.Operations),
			)
		}
		if len(entry.Csps) > 0 {
			fmt.Printf(
				"Note: role %s csps(%d) reserved — menu seed only in this pass\n",
				roleName, len(entry.Csps),
			)
		}
		menus := make([]string, 0, len(entry.Menus))
		for _, menuID := range entry.Menus {
			menuID = strings.TrimSpace(menuID)
			if menuID != "" {
				menus = append(menus, menuID)
			}
		}
		roleMenus[roleName] = menus
	}

	return s.applyRoleMenuPermissionSeed(roleMenus)
}

// initializeMenuPermissionsFromCSVFile 기존 CSV 매트릭스를 적용합니다.
// Deprecated path helper — CSV API 제거 시 함께 삭제 예정.
func (s *MenuService) initializeMenuPermissionsFromCSVFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}
	if len(headers) < 3 {
		return fmt.Errorf("invalid CSV headers in %s", filePath)
	}

	roleNames := headers[2:]
	roleMenus := make(map[string][]string, len(roleNames))
	for _, roleName := range roleNames {
		roleMenus[roleName] = []string{}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}
		if len(record) < 3 {
			continue
		}
		menuID := strings.TrimSpace(record[1])
		if menuID == "" {
			continue
		}
		for i, roleName := range roleNames {
			if i+2 >= len(record) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(record[i+2]), "TRUE") {
				roleMenus[roleName] = append(roleMenus[roleName], menuID)
			}
		}
	}

	return s.applyRoleMenuPermissionSeed(roleMenus)
}

// applyRoleMenuPermissionSeed 역할→메뉴 목록을 DB 매핑으로 upsert(존재 시 skip)합니다.
func (s *MenuService) applyRoleMenuPermissionSeed(roleMenus map[string][]string) error {
	roleIDs := make(map[string]uint, len(roleMenus))
	for roleName := range roleMenus {
		role, err := s.roleRepo.FindRoleByRoleName(roleName, constants.RoleTypePlatform)
		if err != nil {
			return fmt.Errorf("failed to find role %s: %w", roleName, err)
		}
		if role == nil {
			return fmt.Errorf("role not found: %s", roleName)
		}
		roleIDs[roleName] = role.ID
	}

	existingMappings := make(map[string]map[uint]bool)
	for _, roleID := range roleIDs {
		req := &model.MenuMappingFilterRequest{
			RoleIDs: []string{strconv.FormatUint(uint64(roleID), 10)},
		}
		menuIDs, err := s.menuMappingRepo.FindMappedMenuIDs(req)
		if err != nil {
			return fmt.Errorf("failed to get existing mappings for role %d: %w", roleID, err)
		}
		for _, menuID := range menuIDs {
			if _, exists := existingMappings[*menuID]; !exists {
				existingMappings[*menuID] = make(map[uint]bool)
			}
			existingMappings[*menuID][roleID] = true
		}
	}

	for roleName, menus := range roleMenus {
		roleID := roleIDs[roleName]
		for _, menuID := range menus {
			if roleMappings, exists := existingMappings[menuID]; exists && roleMappings[roleID] {
				continue
			}
			mappings := []*model.RoleMenuMapping{
				{RoleID: roleID, MenuID: menuID},
			}
			if err := s.menuRepo.CreateRoleMenuMappings(mappings); err != nil {
				return fmt.Errorf(
					"failed to create menu mapping for %s with role %s: %w",
					menuID, roleName, err,
				)
			}
			if _, exists := existingMappings[menuID]; !exists {
				existingMappings[menuID] = make(map[uint]bool)
			}
			existingMappings[menuID][roleID] = true
		}
	}

	return nil
}

// CreateRoleMenuMappings 역할-메뉴 매핑을 생성합니다
func (s *MenuService) CreateRoleMenuMappings(mappings []*model.RoleMenuMapping) error {
	return s.menuRepo.CreateRoleMenuMappings(mappings)
}

// DeleteRoleMenuMapping 플랫폼 역할-메뉴 매핑 삭제
func (s *MenuService) DeleteRoleMenuMapping(mappings []*model.RoleMenuMapping) error {
	return s.menuRepo.DeleteRoleMenuMapping(mappings)
}

// DeleteRoleMenuMappingByRoleAndMenu role_id + menu_id 조건으로 매핑 삭제
func (s *MenuService) DeleteRoleMenuMappingByRoleAndMenu(roleID uint, menuID string) error {
	return s.menuRepo.DeleteRoleMenuMappingByRoleAndMenu(roleID, menuID)
}

// 해당 role 과 매핑된 메뉴 삭제
func (s *MenuService) DeleteRoleMenuMappingsByRoleID(roleID uint) error {
	return s.menuRepo.DeleteRoleMenuMappingsByRoleID(roleID)
}

const (
	rolePermissionBackupKind     = "role-permission-backup"
	rolePermissionSectionMenus   = "menus"
	rolePermissionSectionOps     = "operations"
	rolePermissionSectionCsps    = "csps"
	rolePermissionRestoreAdd     = "additive"
	rolePermissionRestoreReplace = "replace-role"
)

// BackupRolePermissions 현재 DB의 플랫폼 역할 권한을 백업 문서로 내보냅니다.
// roleNames가 비어 있으면 플랫폼 역할 전체. sections 기본값은 menus.
// role_masters.name 기준으로 직렬화합니다 (numeric role_id 미사용).
func (s *MenuService) BackupRolePermissions(
	roleNames []string, sections []string,
) (*model.RolePermissionBackup, error) {
	sections = normalizeRolePermissionSections(sections)
	roles, err := s.resolvePlatformRolesForBackup(roleNames)
	if err != nil {
		return nil, err
	}

	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Name < roles[j].Name
	})

	entries := make([]model.RolePermissionEntry, 0, len(roles))
	for _, role := range roles {
		entry := model.RolePermissionEntry{
			Role:       role.Name,
			Menus:      []string{},
			Operations: []string{},
			Csps:       []string{},
		}
		if containsSection(sections, rolePermissionSectionMenus) {
			menuIDs, err := s.menuMappingRepo.GetMappedMenuIDs(role.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to list menus for role %s: %w", role.Name, err)
			}
			sort.Strings(menuIDs)
			entry.Menus = menuIDs
		}
		if containsSection(sections, rolePermissionSectionOps) ||
			containsSection(sections, rolePermissionSectionCsps) {
			// reserved: 스키마 자리만 유지 (1단계는 menus)
			fmt.Printf(
				"Note: backup role %s — operations/csps sections reserved (empty)\n",
				role.Name,
			)
		}
		entries = append(entries, entry)
	}

	return &model.RolePermissionBackup{
		Kind:        rolePermissionBackupKind,
		BackupAt:    time.Now().Format(time.RFC3339),
		Source:      "db",
		Sections:    sections,
		Permissions: entries,
	}, nil
}

// SaveRolePermissionBackupFile 백업 문서를 asset/menu/backups/ 에 저장합니다.
func (s *MenuService) SaveRolePermissionBackupFile(
	backup *model.RolePermissionBackup, fileName string,
) (string, error) {
	if backup == nil {
		return "", fmt.Errorf("backup is nil")
	}
	assetPath := util.GetAssetPath()
	dir := filepath.Join(assetPath, "menu", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}
	if fileName == "" {
		fileName = fmt.Sprintf(
			"role-permission-backup-%s.yaml",
			time.Now().Format("20060102-150405"),
		)
	}
	path := filepath.Join(dir, fileName)
	body, err := yaml.Marshal(backup)
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup: %w", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}
	return path, nil
}

// RestoreRolePermissions 역할 권한 백업 문서를 DB에 복구합니다.
// mode: additive | replace-role
func (s *MenuService) RestoreRolePermissions(
	backup *model.RolePermissionBackup, mode string, sections []string,
) (*model.RolePermissionRestoreResult, error) {
	if backup == nil {
		return nil, fmt.Errorf("backup is nil")
	}
	if len(backup.Permissions) == 0 {
		return nil, fmt.Errorf("backup has no permissions entries")
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = rolePermissionRestoreAdd
	}
	if mode != rolePermissionRestoreAdd && mode != rolePermissionRestoreReplace {
		return nil, fmt.Errorf(
			"invalid mode %q (use %s or %s)",
			mode, rolePermissionRestoreAdd, rolePermissionRestoreReplace,
		)
	}
	sections = normalizeRolePermissionSections(sections)
	if backup.Kind != "" && backup.Kind != rolePermissionBackupKind {
		fmt.Printf(
			"Warning: backup kind=%q (expected %q); proceeding\n",
			backup.Kind, rolePermissionBackupKind,
		)
	}

	result := &model.RolePermissionRestoreResult{
		Mode:    mode,
		Message: "role permission restore completed",
	}

	for _, entry := range backup.Permissions {
		roleName := strings.TrimSpace(entry.Role)
		if roleName == "" {
			return nil, fmt.Errorf("permission entry missing role")
		}
		role, err := s.roleRepo.FindRoleByRoleName(roleName, constants.RoleTypePlatform)
		if err != nil {
			return nil, fmt.Errorf("failed to find role %s: %w", roleName, err)
		}
		if role == nil {
			return nil, fmt.Errorf("role not found: %s", roleName)
		}
		result.RolesProcessed++

		if !containsSection(sections, rolePermissionSectionMenus) {
			continue
		}

		desired := uniqueNonEmpty(entry.Menus)
		if mode == rolePermissionRestoreReplace {
			removed, err := s.replaceRoleMenuMappings(role.ID, desired)
			if err != nil {
				return nil, fmt.Errorf("replace-role failed for %s: %w", roleName, err)
			}
			result.MenusRemoved += removed
			result.MenusAdded += len(desired)
			continue
		}

		added, err := s.addMissingRoleMenuMappings(role.ID, desired)
		if err != nil {
			return nil, fmt.Errorf("additive restore failed for %s: %w", roleName, err)
		}
		result.MenusAdded += added
	}

	return result, nil
}

// ParseRolePermissionBackupYAML YAML 바이트를 RolePermissionBackup으로 파싱합니다.
func ParseRolePermissionBackupYAML(body []byte) (*model.RolePermissionBackup, error) {
	var backup model.RolePermissionBackup
	if err := yaml.Unmarshal(body, &backup); err != nil {
		return nil, fmt.Errorf("failed to parse role permission backup: %w", err)
	}
	return &backup, nil
}

// LoadRolePermissionBackupFile 파일에서 백업 문서를 로드합니다.
func (s *MenuService) LoadRolePermissionBackupFile(filePath string) (*model.RolePermissionBackup, error) {
	body, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}
	return ParseRolePermissionBackupYAML(body)
}

func (s *MenuService) resolvePlatformRolesForBackup(
	roleNames []string,
) ([]*model.RoleMaster, error) {
	if len(roleNames) > 0 {
		out := make([]*model.RoleMaster, 0, len(roleNames))
		for _, name := range roleNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			role, err := s.roleRepo.FindRoleByRoleName(name, constants.RoleTypePlatform)
			if err != nil {
				return nil, fmt.Errorf("failed to find role %s: %w", name, err)
			}
			if role == nil {
				return nil, fmt.Errorf("role not found: %s", name)
			}
			out = append(out, role)
		}
		return out, nil
	}

	roles, err := s.roleRepo.FindRoles(&model.RoleFilterRequest{
		RoleTypes: []constants.IAMRoleType{constants.RoleTypePlatform},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list platform roles: %w", err)
	}
	return roles, nil
}

func (s *MenuService) addMissingRoleMenuMappings(roleID uint, menuIDs []string) (int, error) {
	existing, err := s.menuMappingRepo.GetMappedMenuIDs(roleID)
	if err != nil {
		return 0, err
	}
	have := make(map[string]bool, len(existing))
	for _, id := range existing {
		have[id] = true
	}
	added := 0
	for _, menuID := range menuIDs {
		if have[menuID] {
			continue
		}
		mapping := &model.RoleMenuMapping{RoleID: roleID, MenuID: menuID}
		if err := s.menuRepo.CreateRoleMenuMappings([]*model.RoleMenuMapping{mapping}); err != nil {
			return added, err
		}
		added++
		have[menuID] = true
	}
	return added, nil
}

func (s *MenuService) replaceRoleMenuMappings(roleID uint, menuIDs []string) (int, error) {
	existing, err := s.menuMappingRepo.GetMappedMenuIDs(roleID)
	if err != nil {
		return 0, err
	}
	if err := s.menuRepo.DeleteRoleMenuMappingsByRoleID(roleID); err != nil {
		return 0, err
	}
	if len(menuIDs) > 0 {
		mappings := make([]*model.RoleMenuMapping, 0, len(menuIDs))
		for _, menuID := range menuIDs {
			mappings = append(mappings, &model.RoleMenuMapping{
				RoleID: roleID,
				MenuID: menuID,
			})
		}
		if err := s.menuRepo.CreateRoleMenuMappings(mappings); err != nil {
			return 0, err
		}
	}
	return len(existing), nil
}

func normalizeRolePermissionSections(sections []string) []string {
	if len(sections) == 0 {
		return []string{rolePermissionSectionMenus}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(sections))
	for _, s := range sections {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" || seen[s] {
			continue
		}
		switch s {
		case rolePermissionSectionMenus, rolePermissionSectionOps, rolePermissionSectionCsps:
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return []string{rolePermissionSectionMenus}
	}
	return out
}

func containsSection(sections []string, target string) bool {
	for _, s := range sections {
		if s == target {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
