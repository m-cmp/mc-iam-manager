package service

import (
	"fmt"
	"io"       // Added
	"net/http" // Added
	"os"       // Added
	"path/filepath"
	"strings" // Added

	"github.com/joho/godotenv" // Added
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gopkg.in/yaml.v3" // Use v3
)

type MenuService struct {
	menuRepo *repository.MenuRepository
}

func NewMenuService(menuRepo *repository.MenuRepository) *MenuService {
	return &MenuService{
		menuRepo: menuRepo,
	}
}

// GetMenus 모든 메뉴 조회
func (s *MenuService) GetMenus() ([]model.Menu, error) {
	return s.menuRepo.GetMenus()
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
		envPath := filepath.Join("..", ".env") // Path relative to src directory
		_ = godotenv.Load(envPath)             // Ignore error if .env not found
		menuURL := os.Getenv("MCWEBCONSOLE_MENUYAML")

		// Default local path if URL is not set or not a valid URL
		defaultLocalPath := filepath.Join("..", "asset", "menu", "menu.yaml")

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
			// If MCWEBCONSOLE_MENUYAML is set but not a URL, assume it's a local path
			fmt.Printf("Using local menu YAML path from MCWEBCONSOLE_MENUYAML: %s\n", menuURL)
			// Note: This path should be relative to the project root or absolute.
			// Adjusting relative path from .env based on current execution dir (src) might be needed.
			// Assuming menuURL is relative to project root:
			effectiveFilePath = filepath.Join("..", menuURL)
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
