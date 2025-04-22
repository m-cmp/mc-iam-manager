package service

import (
	"context" // Added
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort" // Added
	"strings"

	"github.com/joho/godotenv"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm" // Import gorm
)

type MenuService struct {
	db             *gorm.DB // Add db field
	menuRepo       *repository.MenuRepository
	userRepo       *repository.UserRepository       // Added dependency
	permissionRepo *repository.PermissionRepository // Added dependency
}

// NewMenuService 새 MenuService 인스턴스 생성
func NewMenuService(db *gorm.DB) *MenuService { // Accept only db
	// Initialize repositories internally
	menuRepo := repository.NewMenuRepository(db)
	userRepo := repository.NewUserRepository(db)
	permissionRepo := repository.NewPermissionRepository() // Permission repo doesn't need db in constructor
	return &MenuService{
		db:             db, // Store db
		menuRepo:       menuRepo,
		userRepo:       userRepo,
		permissionRepo: permissionRepo,
	}
}

// GetAllMenusTree 모든 메뉴를 트리 구조로 조회 (관리자용)
func (s *MenuService) GetAllMenusTree() ([]*model.MenuTreeNode, error) {
	allMenus, err := s.menuRepo.GetMenus() // Get all menus
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

// BuildUserMenuTree 사용자의 Platform Role 기반 메뉴 트리 조회 (내부 로직용)
func (s *MenuService) BuildUserMenuTree(ctx context.Context, kcUserID string) ([]*model.MenuTreeNode, error) {
	// 1. Get User's Platform Roles
	// Assuming userID is Keycloak ID, fetch user details including roles
	// Note: userRepo.GetUserByID currently only fetches basic info + roles from DB based on kc_id.
	// Ensure userRepo.GetUserByID correctly preloads PlatformRoles.
	user, err := s.userRepo.FindByKcID(kcUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}
	if user == nil || user.PlatformRoles == nil || len(user.PlatformRoles) == 0 {
		return []*model.MenuTreeNode{}, nil // No roles, no menus
	}

	platformRoleIDs := make([]uint, len(user.PlatformRoles))
	for i, role := range user.PlatformRoles {
		platformRoleIDs[i] = role.ID
	}

	// 2. Get Permissions for those Roles
	// Need a method in PermissionRepository like GetPermissionsByRoleIDs(ctx, roleType, roleIDs)
	// For now, iterate and call GetRolePermissions (less efficient)
	allowedMenuIDs := make(map[string]bool)
	tx := s.db.WithContext(ctx) // Create transaction with context
	for _, roleID := range platformRoleIDs {
		permissions, err := s.permissionRepo.GetRolePermissions(tx, "platform", roleID) // Pass tx
		if err != nil {
			// Log error but continue, maybe user has other roles with permissions
			fmt.Printf("Warning: failed to get permissions for platform role %d: %v\n", roleID, err)
			continue
		}
		for _, p := range permissions {
			// Assuming permission ID directly corresponds to menu ID for menu permissions
			allowedMenuIDs[p.ID] = true
		}
	}

	if len(allowedMenuIDs) == 0 {
		return []*model.MenuTreeNode{}, nil // No menu permissions
	}

	// 3. Get All Menus from DB
	allMenus, err := s.menuRepo.GetMenus() // Get the flat list
	if err != nil {
		return nil, fmt.Errorf("failed to get all menus: %w", err)
	}
	if len(allMenus) == 0 {
		return []*model.MenuTreeNode{}, nil
	}

	// 4. Build Menu Map for easy lookup
	menuMap := make(map[string]*model.Menu, len(allMenus))
	for i := range allMenus {
		menuMap[allMenus[i].ID] = &allMenus[i]
	}

	// 5. Determine Full Set of Accessible Menus (including parents)
	accessibleMenuIDs := make(map[string]bool)
	for menuID := range allowedMenuIDs {
		currID := menuID
		for {
			if _, visited := accessibleMenuIDs[currID]; visited {
				break // Already processed this branch
			}
			menu, exists := menuMap[currID]
			if !exists {
				break // Parent ID doesn't exist in the map (data inconsistency?)
			}
			accessibleMenuIDs[currID] = true
			if menu.ParentID == "" {
				break // Reached root
			}
			currID = menu.ParentID
		}
	}

	// 6. Filter All Menus based on accessibility
	accessibleMenus := make([]model.Menu, 0, len(accessibleMenuIDs))
	for _, menu := range allMenus {
		if accessibleMenuIDs[menu.ID] {
			accessibleMenus = append(accessibleMenus, menu)
		}
	}

	// 7. Build the Tree Structure
	tree := buildMenuTree(accessibleMenus)

	return tree, nil
}

// buildMenuTree 재귀적으로 메뉴 트리 구조 생성 (헬퍼 함수)
func buildMenuTree(menus []model.Menu) []*model.MenuTreeNode {
	nodeMap := make(map[string]*model.MenuTreeNode, len(menus))
	var rootNodes []*model.MenuTreeNode

	// Create all nodes and put them in a map
	for i := range menus {
		node := &model.MenuTreeNode{
			Menu: menus[i], // Copy menu data
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
func (s *MenuService) GetMenus() ([]model.Menu, error) {
	// return s.menuRepo.GetMenus() // Keep original GetMenus for now
	var menus []model.Menu
	menus, err := s.menuRepo.GetMenus()
	return menus, err
}

// GetByID 메뉴 ID로 조회
func (s *MenuService) GetByID(id string) (*model.Menu, error) {
	return s.menuRepo.GetByID(id)
}

// Create 새 메뉴 생성
func (s *MenuService) Create(menu *model.Menu) error {
	// TODO: 필요한 비즈니스 로직 추가 (예: 유효성 검사)
	return s.menuRepo.Create(menu)
}

// Update 메뉴 정보 부분 업데이트
func (s *MenuService) Update(id string, updates map[string]interface{}) error {
	// TODO: 필요한 비즈니스 로직 추가 (예: 유효성 검사, 업데이트 가능 필드 제한 등)

	// 업데이트 전 레코드 존재 여부 확인 (선택 사항, Repository에서도 확인)
	// _, err := s.GetByID(id)
	// if err != nil {
	// 	 return err
	// }

	return s.menuRepo.Update(id, updates)
}

// Delete 메뉴 삭제
func (s *MenuService) Delete(id string) error {
	// TODO: 필요한 비즈니스 로직 추가 (예: 하위 메뉴 처리 등)
	return s.menuRepo.Delete(id)
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

	// 2. DB에 Upsert
	if err := s.menuRepo.UpsertMenus(menus); err != nil {
		return fmt.Errorf("failed to upsert menus to database: %w", err)
	}

	// log.Printf("Successfully registered %d menus from YAML file %s", len(menus), effectiveFilePath)
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
