package service

import (
	"context" // Added
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort" // Added
	"strconv"
	"strings"

	"encoding/csv"

	"github.com/joho/godotenv"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm" // Import gorm
)

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
		MenuID: menuIDs,
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
		MenuID: parentIDs,
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
		MenuID: menuIDs,
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
		MenuID: parentIDs,
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
	// TODO: 필요한 비즈니스 로직 추가 (예: 유효성 검사)
	return s.menuRepo.CreateMenu(req)
}

// Update 메뉴 정보 부분 업데이트
func (s *MenuService) Update(id string, updates map[string]interface{}) error {
	// TODO: 필요한 비즈니스 로직 추가 (예: 유효성 검사, 업데이트 가능 필드 제한 등)

	// 업데이트 전 레코드 존재 여부 확인 (선택 사항, Repository에서도 확인)
	// _, err := s.GetByID(id)
	// if err != nil {
	// 	 return err
	// }

	return s.menuRepo.UpdateMenu(id, updates)
}

// Delete 메뉴 삭제
func (s *MenuService) Delete(id string) error {
	// TODO: 필요한 비즈니스 로직 추가 (예: 하위 메뉴 처리 등)
	return s.menuRepo.DeleteMenu(id)
}

// LoadAndRegisterMenusFromYAML YAML 파일에서 메뉴를 로드하여 DB에 등록(Upsert)
// filePath 쿼리 파라미터가 없으면 .env의 MCWEBCONSOLE_MENUYAML URL에서 다운로드 시도
func (s *MenuService) LoadAndRegisterMenusFromYAML(filePath string) error {
	effectiveFilePath := filePath
	downloaded := false

	// If filePath is not provided via query param
	if effectiveFilePath == "" {
		// Load .env file to get the URL (assuming .env is at project root)
		// .env path should be relative to project root when running the binary
		envPath := ".env"          // Path relative to project root
		_ = godotenv.Load(envPath) // Ignore error if .env not found
		menuURL := os.Getenv("MCWEBCONSOLE_MENUYAML")

		// Default local path relative to project root
		defaultLocalPath := filepath.Join("asset", "menu", "menu.yaml") // Removed "../"

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
			// If MCWEBCONSOLE_MENUYAML is set but not a URL, assume it's a local path relative to project root
			fmt.Printf("Using local menu YAML path from MCWEBCONSOLE_MENUYAML: %s\n", menuURL)
			// Assuming menuURL is relative to project root:
			effectiveFilePath = menuURL // Use the path directly
		} else {
			// If MCWEBCONSOLE_MENUYAML is not set, use the default local path
			fmt.Printf("MCWEBCONSOLE_MENUYAML not set or invalid URL, using default local path: %s\n", defaultLocalPath)
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
			if err := s.menuRepo.UpdateMenu("home", map[string]interface{}{
				"display_name": menu.DisplayName,
				"res_type":     menu.ResType,
				"is_action":    menu.IsAction,
				"priority":     menu.Priority,
				"menu_number":  menu.MenuNumber,
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
		if menu.ID != "home" {
			menusToUpsert = append(menusToUpsert, menu)
		}
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
	for _, roleID := range req.RoleID {
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

// CreateMenuMapping 메뉴 매핑을 생성합니다
func (s *MenuService) CreateMenuMappings(mappings []*model.MenuMapping) error {
	return s.menuRepo.CreateMenuMappings(mappings)
}

// DeleteMenuMapping 플랫폼 역할-메뉴 매핑 삭제
func (s *MenuService) DeleteMenuMapping(platformRoleID uint, menuID string) error {
	return s.menuMappingRepo.DeleteMapping(platformRoleID, menuID)
}

// InitializeMenuPermissionsFromCSV CSV 파일을 읽어서 메뉴 권한을 초기화합니다.
// filePath 쿼리 파라미터가 없으면 기본 경로인 asset/menu/permission.csv를 사용
func (s *MenuService) InitializeMenuPermissionsFromCSV(filePath string) error {
	effectiveFilePath := filePath

	// If filePath is not provided via query param
	if effectiveFilePath == "" {
		// Load .env file to get the URL (assuming .env is at project root)
		envPath := ".env"          // Path relative to project root
		_ = godotenv.Load(envPath) // Ignore error if .env not found
		permissionURL := os.Getenv("MCWEBCONSOLE_PERMISSIONCSV")

		// Default local path relative to project root
		defaultLocalPath := filepath.Join("asset", "menu", "permission.csv")

		if permissionURL != "" && (strings.HasPrefix(permissionURL, "http://") || strings.HasPrefix(permissionURL, "https://")) {
			// Attempt to download from URL
			fmt.Printf("Attempting to download permission CSV from URL: %s\n", permissionURL)
			resp, err := http.Get(permissionURL)
			if err != nil {
				fmt.Printf("Warning: Failed to download permission CSV from %s: %v. Falling back to local path: %s\n", permissionURL, err, defaultLocalPath)
				effectiveFilePath = defaultLocalPath
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					fmt.Printf("Warning: Failed to download permission CSV from %s (Status: %s). Falling back to local path: %s\n", permissionURL, resp.Status, defaultLocalPath)
					effectiveFilePath = defaultLocalPath
				} else {
					bodyBytes, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Printf("Warning: Failed to read response body from %s: %v. Falling back to local path: %s\n", permissionURL, err, defaultLocalPath)
						effectiveFilePath = defaultLocalPath
					} else {
						// Create a temporary file
						tempFile, err := os.CreateTemp("", "permission-*.csv")
						if err != nil {
							fmt.Printf("Warning: Failed to create temp file: %v. Falling back to local path: %s\n", err, defaultLocalPath)
							effectiveFilePath = defaultLocalPath
						} else {
							defer os.Remove(tempFile.Name()) // Clean up temp file when done
							if _, err := tempFile.Write(bodyBytes); err != nil {
								fmt.Printf("Warning: Failed to write to temp file: %v. Falling back to local path: %s\n", err, defaultLocalPath)
								effectiveFilePath = defaultLocalPath
							} else {
								effectiveFilePath = tempFile.Name()
							}
						}
					}
				}
			}
		} else {
			effectiveFilePath = defaultLocalPath
		}
	}

	// CSV 파일 열기
	file, err := os.Open(effectiveFilePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// CSV 리더 생성
	reader := csv.NewReader(file)

	// 첫 번째 행 읽기 (헤더)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// 역할 이름 목록 추출 (framework, resource 이후의 컬럼들)
	roleNames := headers[2:]

	// DB에서 역할 ID 조회
	roleIDs := make(map[string]uint)
	for _, roleName := range roleNames {
		role, err := s.roleRepo.FindRoleByRoleName(roleName, constants.RoleTypePlatform)
		if err != nil {
			return fmt.Errorf("failed to find role %s: %w", roleName, err)
		}
		if role == nil {
			return fmt.Errorf("role not found: %s", roleName)
		}
		roleIDs[roleName] = role.ID
	}

	// 기존 매핑 조회
	existingMappings := make(map[string]map[uint]bool) // menuID -> roleID -> exists
	for _, roleID := range roleIDs {
		req := &model.MenuMappingFilterRequest{
			RoleID: []string{strconv.FormatUint(uint64(roleID), 10)},
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

	// 나머지 행 읽기
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		// framework와 resource 추출
		// framework := record[0]
		// resource := record[1]
		// 메뉴 ID 생성 (framework:resource 형식)
		//menuID := fmt.Sprintf("%s:%s", framework, resource)
		menuID := record[1] // menuId 그대로 넣도록 한다.

		// 각 역할별 권한 확인
		for i, roleName := range roleNames {
			hasPermission := record[i+2] == "TRUE" // +2 because first two columns are framework and resource

			if hasPermission {
				roleID := roleIDs[roleName]

				// 중복 매핑 체크
				if roleMappings, exists := existingMappings[menuID]; exists {
					if roleMappings[roleID] {
						// 이미 매핑이 존재하면 스킵
						continue
					}
				}

				// 새 매핑 생성
				err := s.menuMappingRepo.CreateMapping(roleID, menuID)
				if err != nil {
					return fmt.Errorf("failed to create menu mapping for %s with role %s: %w", menuID, roleName, err)
				}

				// 매핑 정보 업데이트
				if _, exists := existingMappings[menuID]; !exists {
					existingMappings[menuID] = make(map[uint]bool)
				}
				existingMappings[menuID][roleID] = true
			}
		}
	}

	return nil
}

// CreateRoleMenuMappings 역할-메뉴 매핑을 생성합니다
func (s *MenuService) CreateRoleMenuMappings(mappings []*model.RoleMenuMapping) error {
	return s.menuRepo.CreateRoleMenuMappings(mappings)
}
