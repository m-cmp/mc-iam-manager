package service

import (
	"context"       // Add context import
	"encoding/json" // Add json import for request body
	"errors"

	// "errors"        // Remove unused import
	"fmt"     // Add fmt import for errors
	"log"     // Add log import
	"os"      // Import os package to read environment variables
	"strings" // Add strings import for string operations

	// "net/http"      // Remove unused import

	// Import godotenv for loading environment variables
	"github.com/m-cmp/mc-iam-manager/model" // Import mcmpapi model
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// ProjectService 프로젝트 관리 서비스
type ProjectService struct {
	db             *gorm.DB
	projectRepo    *repository.ProjectRepository
	workspaceRepo  *repository.WorkspaceRepository
	mcmpApiService McmpApiService // Added dependency back
}

// NewProjectService 새 ProjectService 인스턴스 생성
func NewProjectService(db *gorm.DB) *ProjectService {
	log.Printf("Creating new ProjectService with db: %+v", db)

	mcmpApiService := NewMcmpApiService(db)
	log.Printf("Created mcmpApiService: %+v", mcmpApiService)

	projectService := &ProjectService{
		db:             db,
		projectRepo:    repository.NewProjectRepository(db),
		workspaceRepo:  repository.NewWorkspaceRepository(db),
		mcmpApiService: mcmpApiService,
	}

	log.Printf("Created ProjectService with mcmpApiService: %+v", projectService.mcmpApiService)
	return projectService
}

// Create 프로젝트 생성 (mc-infra-manager 호출 및 DB 저장)
// workspaceID is optional: if 0, assigns to default workspace; otherwise assigns to specified workspace
func (s *ProjectService) Create(ctx context.Context, project *model.Project, workspaceID ...uint) error {
	// 이름 중복 체크
	existingProject, err := s.projectRepo.FindProjectByProjectName(project.Name)
	if err == nil && existingProject != nil {
		return fmt.Errorf("project with name '%s' already exists", project.Name)
	}
	if err != nil && err.Error() != "project not found" {
		return err
	}

	// Step 0: Determine target workspace (specified or default) and validate BEFORE calling mc-infra-manager
	var targetWorkspace *model.Workspace
	var targetWorkspaceID uint

	if len(workspaceID) > 0 && workspaceID[0] != 0 {
		// Workspace specified, verify it exists
		targetWorkspaceID = workspaceID[0]
		targetWorkspace, err = s.workspaceRepo.FindWorkspaceByID(targetWorkspaceID)
		if err != nil || targetWorkspace == nil {
			log.Printf("Error finding specified workspace %d: %v", targetWorkspaceID, err)
			return fmt.Errorf("workspace not found")
		}
		log.Printf("Validated specified workspace: %s (ID: %d)", targetWorkspace.Name, targetWorkspace.ID)
	} else {
		// No workspace specified, use default
		defaultWsName := os.Getenv("DEFAULT_WORKSPACE_NAME")
		if defaultWsName == "" {
			defaultWsName = "default"
			log.Printf("DEFAULT_WORKSPACE_NAME not set in environment, using default value: %s", defaultWsName)
		}
		log.Printf("Using default workspace name: %s", defaultWsName)
		targetWorkspace, err = s.workspaceRepo.FindWorkspaceByName(defaultWsName)
		if err != nil {
			if err.Error() == "workspace not found" {
				// Default workspace doesn't exist, create it
				log.Printf("Default workspace '%s' not found. Creating it...", defaultWsName)
				newWorkspace := &model.Workspace{
					Name:        defaultWsName,
					Description: "Default workspace for automatically synced projects",
				}
				if err := s.workspaceRepo.CreateWorkspace(newWorkspace); err != nil {
					log.Printf("Error creating default workspace '%s': %v", defaultWsName, err)
					return fmt.Errorf("failed to create default workspace: %w", err)
				}
				log.Printf("Successfully created default workspace '%s'", defaultWsName)
				targetWorkspace = newWorkspace
			} else {
				log.Printf("Error finding default workspace '%s': %v. Cannot assign projects.", defaultWsName, err)
				return fmt.Errorf("failed to find or create default workspace: %w", err)
			}
		}
		targetWorkspaceID = targetWorkspace.ID
	}
	log.Printf("Workspace validation complete. Will assign project to workspace: %s (ID: %d)", targetWorkspace.Name, targetWorkspaceID)

	log.Printf("Attempting to create namespace in mc-infra-manager for project: %s", project.Name)

	// Check if mcmpApiService is properly initialized
	if s.mcmpApiService == nil {
		log.Printf("ERROR: mcmpApiService is nil! This indicates a configuration issue.")
		return fmt.Errorf("mcmpApiService is not properly initialized")
	}

	// 1. Call mc-infra-manager PostNs API
	nsRequestBody := map[string]string{
		"name":        project.Name,
		"description": project.Description,
	}

	callReq := &model.McmpApiCallRequest{
		ServiceName: "mc-infra-manager",
		ActionName:  "Postns",
		RequestParams: model.McmpApiRequestParams{
			Body: nsRequestBody,
		},
	}

	log.Printf("About to call mcmpApiService.McmpApiCall with service: %+v", s.mcmpApiService)
	statusCode, respBody, serviceVersion, calledURL, err := s.mcmpApiService.McmpApiCall(ctx, callReq)
	if err != nil {
		log.Printf("Error calling %s(v%s) %s (URL: %s): %v (status code: %d)", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, err, statusCode)
		return fmt.Errorf("failed to call %s(v%s) %s (URL: %s): %w (status code: %d)", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, err, statusCode)
	}
	if statusCode < 200 || statusCode >= 300 {
		log.Printf("%s(v%s) %s call failed (URL: %s): status code %d, response: %s", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, statusCode, string(respBody))
		var errorResp map[string]interface{}
		errMsg := fmt.Sprintf("%s(v%s) %s call failed with status code %d (URL: %s)", callReq.ServiceName, serviceVersion, callReq.ActionName, statusCode, calledURL)
		if json.Unmarshal(respBody, &errorResp) == nil {
			if msg, ok := errorResp["message"].(string); ok {
				errMsg = fmt.Sprintf("%s(v%s) error: %s (URL: %s, Status: %d)", callReq.ServiceName, serviceVersion, msg, calledURL, statusCode)
			}
		}
		return errors.New(errMsg)
	}

	// Extract NsId from response
	var nsResponse map[string]interface{}
	if jsonErr := json.Unmarshal(respBody, &nsResponse); jsonErr == nil {
		if nsIdVal, ok := nsResponse["id"].(string); ok {
			project.NsId = nsIdVal
			log.Printf("Assigned NsId %s from mc-infra-manager response", project.NsId)
		} else {
			log.Printf("Warning: 'id' field not found or not a string in PostNs response: %+v", nsResponse)
		}
	} else {
		log.Printf("Warning: could not parse JSON response from PostNs: %v. Response body: %s", jsonErr, string(respBody))
	}
	// Fallback if NsId is still empty
	if project.NsId == "" {
		log.Printf("Warning: NsId is empty after PostNs call, using project name as fallback.")
		project.NsId = project.Name
	}

	log.Printf("Successfully called mc-infra-manager PostNs. Proceeding to create project in local DB: %+v", project)

	// 2. Create project in local DB
	if err := s.projectRepo.CreateProject(project); err != nil {
		return err
	}

	// 3. Assign project to target workspace (already validated above)
	if err := s.projectRepo.AddProjectWorkspaceAssociation(project.ID, targetWorkspaceID); err != nil {
		log.Printf("Error assigning project %d to workspace %d: %v", project.ID, targetWorkspaceID, err)
		return nil // Project created but assignment failed
	}
	log.Printf("Successfully assigned project %d to workspace %d (%s)", project.ID, targetWorkspaceID, targetWorkspace.Name)

	return nil
}

// List 모든 프로젝트 조회
func (s *ProjectService) ListProjects(req *model.ProjectFilterRequest) ([]*model.Project, error) {
	return s.projectRepo.FindProjects(req)
}

// GetProjectWorkspaces 프로젝트에 할당된 workspace 목록 조회
func (s *ProjectService) GetProjectWorkspaces(projectID uint) ([]*model.Workspace, error) {
	// 프로젝트 존재 여부 확인
	_, err := s.projectRepo.FindProjectByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	// 할당된 workspace 목록 조회
	return s.projectRepo.FindAssignedWorkspaces(projectID)
}

// GetByID ID로 프로젝트 조회
func (s *ProjectService) GetProjectByID(id uint) (*model.Project, error) {
	return s.projectRepo.FindProjectByProjectID(id)
}

// GetByName 이름으로 프로젝트 조회
func (s *ProjectService) GetProjectByName(name string) (*model.Project, error) {
	return s.projectRepo.FindProjectByProjectName(name)
}

// Update 프로젝트 정보 부분 업데이트
func (s *ProjectService) UpdateProject(id uint, updates map[string]interface{}) error {
	_, err := s.projectRepo.FindProjectByProjectID(id)
	if err != nil {
		// Propagate the error (e.g., ErrProjectNotFound)
		return err
	}
	return s.projectRepo.UpdateProject(id, updates)
}

// Delete 프로젝트 삭제
func (s *ProjectService) DeleteProject(id uint) error {
	// 1. 프로젝트 존재 여부 확인
	project, err := s.projectRepo.FindProjectByProjectID(id)
	if err != nil {
		return err
	}

	// 2. 워크스페이스 할당 확인
	workspaces, err := s.projectRepo.FindAssignedWorkspaces(id)
	if err != nil {
		return fmt.Errorf("워크스페이스 할당 확인 실패: %v", err)
	}

	// 3. 할당된 워크스페이스가 있으면 삭제 불가
	if len(workspaces) > 0 {
		workspaceNames := make([]string, len(workspaces))
		for i, ws := range workspaces {
			workspaceNames[i] = ws.Name
		}
		return fmt.Errorf("프로젝트가 워크스페이스에 할당되어 있습니다: %s. 먼저 모든 워크스페이스에서 할당을 해제하세요",
			strings.Join(workspaceNames, ", "))
	}

	// 4. mc-infra-manager namespace 삭제
	if project.NsId != "" {
		ctx := context.Background()
		callReq := &model.McmpApiCallRequest{
			ServiceName: "mc-infra-manager",
			ActionName:  "DeleteNs",
			RequestParams: model.McmpApiRequestParams{
				PathParams: map[string]string{
					"nsId": project.NsId,
				},
			},
		}

		statusCode, respBody, _, _, err := s.mcmpApiService.McmpApiCall(ctx, callReq)
		if err != nil {
			log.Printf("Warning: failed to delete namespace %s from mc-infra-manager: %v", project.NsId, err)
			// 계속 진행 (DB 정리는 수행)
		} else if statusCode < 200 || statusCode >= 300 {
			log.Printf("Warning: mc-infra-manager DeleteNs failed (status %d): %s", statusCode, string(respBody))
			// 계속 진행 (DB 정리는 수행)
		} else {
			log.Printf("Successfully deleted namespace %s from mc-infra-manager", project.NsId)
		}
	}

	// 5. DB에서 프로젝트 삭제
	return s.projectRepo.DeleteProject(id)
}

// AddWorkspaceAssociation 프로젝트에 워크스페이스 연결
func (s *ProjectService) AddWorkspaceAssociation(projectID, workspaceID uint) error {
	// Check if both project and workspace exist
	_, errPr := s.projectRepo.FindProjectByProjectID(projectID)
	if errPr != nil {
		return errPr
	}
	_, errWs := s.workspaceRepo.FindWorkspaceByID(workspaceID)
	if errWs != nil {
		return errWs
	}
	return s.projectRepo.AddProjectWorkspaceAssociation(projectID, workspaceID)
}

// SyncProjectsWithInfraManager mc-infra-manager의 네임스페이스와 로컬 프로젝트 동기화
func (s *ProjectService) SyncProjectsWithInfraManager(ctx context.Context) error {
	log.Println("Starting project synchronization with mc-infra-manager...")

	// Check if mcmpApiService is properly initialized
	if s.mcmpApiService == nil {
		log.Printf("ERROR: mcmpApiService is nil! This indicates a configuration issue.")
		return fmt.Errorf("mcmpApiService is not properly initialized")
	}

	// 1. Call mc-infra-manager GetAllNs API
	callReq := &model.McmpApiCallRequest{
		ServiceName: "mc-infra-manager",
		ActionName:  "GetAllNs",
		RequestParams: model.McmpApiRequestParams{ // No params needed for GetAllNs
			PathParams:  nil,
			QueryParams: nil,
			Body:        nil,
		},
	}

	log.Printf("About to call mcmpApiService.McmpApiCall with service: %+v", s.mcmpApiService)
	statusCode, respBody, serviceVersion, calledURL, err := s.mcmpApiService.McmpApiCall(ctx, callReq)
	if err != nil {
		log.Printf("Error calling %s(v%s) %s (URL: %s): %v (status code: %d)", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, err, statusCode)
		return fmt.Errorf("failed to call mc-infra-manager GetAllNs: %w (status code: %d)", err, statusCode)
	}
	if statusCode < 200 || statusCode >= 300 {
		log.Printf("%s(v%s) %s call failed (URL: %s): status code %d, response: %s", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, statusCode, string(respBody))
		return fmt.Errorf("mc-infra-manager GetAllNs call failed with status code %d", statusCode)
	}

	// 2. Parse response and extract namespaces
	var infraResponse struct { // Define a struct to parse the expected response
		Ns []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"ns"`
	}
	if err := json.Unmarshal(respBody, &infraResponse); err != nil {
		log.Printf("Error unmarshalling mc-infra-manager GetAllNs response: %v. Body: %s", err, string(respBody))
		return fmt.Errorf("failed to parse response from mc-infra-manager: %w", err)
	}

	if len(infraResponse.Ns) == 0 {
		log.Println("No namespaces found in mc-infra-manager. Synchronization finished.")
		return nil
	}
	log.Printf("Found %d namespaces in mc-infra-manager.", len(infraResponse.Ns))

	// 3. Get local projects
	localProjects, err := s.projectRepo.FindProjects(&model.ProjectFilterRequest{})
	if err != nil {
		log.Printf("Error listing local projects: %v", err)
		return fmt.Errorf("failed to list local projects: %w", err)
	}

	// 4. Create a map of existing local projects by NsId for quick lookup
	localProjectMap := make(map[string]bool)
	for _, p := range localProjects {
		if p.NsId != "" {
			localProjectMap[p.NsId] = true
		}
	}
	log.Printf("Found %d local projects with NsId.", len(localProjectMap))

	// Get all project-workspace assignments
	assignedProjectMap, err := s.projectRepo.FindAllProjectWorkspaceAssignments()
	if err != nil {
		log.Printf("Error getting project workspace assignments: %v", err)
		return fmt.Errorf("failed to get project assignments: %w", err)
	}
	log.Printf("Found %d projects assigned to at least one workspace.", len(assignedProjectMap))

	// Get default workspace ID once
	defaultWsName := os.Getenv("DEFAULT_WORKSPACE_NAME")
	if defaultWsName == "" {
		defaultWsName = "default"
		log.Printf("DEFAULT_WORKSPACE_NAME not set in environment, using default value: %s", defaultWsName)
	}
	log.Printf("Using workspace name: %s", defaultWsName)
	defaultWs, err := s.workspaceRepo.FindWorkspaceByName(defaultWsName)
	if err != nil {
		if err.Error() == "workspace not found" {
			// Default workspace doesn't exist, create it
			log.Printf("Default workspace '%s' not found. Creating it...", defaultWsName)
			newWorkspace := &model.Workspace{
				Name:        defaultWsName,
				Description: "Default workspace for automatically synced projects",
			}
			if err := s.workspaceRepo.CreateWorkspace(newWorkspace); err != nil {
				log.Printf("Error creating default workspace '%s': %v", defaultWsName, err)
				return fmt.Errorf("failed to create default workspace: %w", err)
			}
			log.Printf("Successfully created default workspace '%s'", defaultWsName)
			defaultWs = newWorkspace
		} else {
			log.Printf("Error finding default workspace '%s': %v. Cannot assign projects.", defaultWsName, err)
			return fmt.Errorf("failed to find or create default workspace: %w", err)
		}
	}

	// 5. Compare, create missing projects, and assign unassigned existing projects
	addedCount := 0
	assignedToDefaultCount := 0
	for _, infraNs := range infraResponse.Ns {
		var currentProjectID uint
		var isNewProject bool

		// Check if project exists locally based on NsId
		var existingProject *model.Project
		for _, p := range localProjects {
			if p.NsId == infraNs.ID {
				existingProject = p
				break
			}
		}

		if existingProject == nil {
			// Project does not exist locally, create it
			isNewProject = true
			log.Printf("Namespace '%s' (ID: %s) not found locally. Creating project...", infraNs.Name, infraNs.ID)
			newProject := &model.Project{
				NsId:        infraNs.ID,
				Name:        infraNs.Name, // Use infra name as local name
				Description: infraNs.Description,
			}
			if err := s.projectRepo.CreateProject(newProject); err != nil {
				// Log the error but continue syncing other projects
				log.Printf("Error creating project for namespace '%s' (ID: %s): %v", infraNs.Name, infraNs.ID, err)
				continue // Skip assignment if creation failed
			}
			log.Printf("Successfully created project for namespace '%s' (ID: %s)", infraNs.Name, infraNs.ID)
			addedCount++
			currentProjectID = newProject.ID // Use the ID of the newly created project
		} else {
			// Project already exists locally
			isNewProject = false
			currentProjectID = existingProject.ID
			log.Printf("Project for namespace '%s' (ID: %s) already exists locally (Project ID: %d). Checking assignment...", infraNs.Name, infraNs.ID, currentProjectID)
		}

		// Check if the project (new or existing) is assigned to any workspace
		if _, isAssigned := assignedProjectMap[currentProjectID]; !isAssigned {
			// Project is not assigned to any workspace, assign to default
			log.Printf("Project %d (NsId: %s) is not assigned to any workspace. Assigning to default workspace %d...", currentProjectID, infraNs.ID, defaultWs.ID)
			if assignErr := s.projectRepo.AddProjectWorkspaceAssociation(currentProjectID, defaultWs.ID); assignErr != nil {
				log.Printf("Error assigning project %d to default workspace %d: %v", currentProjectID, defaultWs.ID, assignErr)
			} else {
				log.Printf("Successfully assigned project %d to default workspace %d", currentProjectID, defaultWs.ID)
				if !isNewProject { // Count only assignments for existing projects here
					assignedToDefaultCount++
				}
				// Add to map immediately to avoid re-checking if somehow processed again (though unlikely with current loop)
				assignedProjectMap[currentProjectID] = true
			}
		} else if isNewProject {
			// This case should ideally not happen if GetAllProjectWorkspaceAssignments is correct,
			// but log a warning if a newly created project ID somehow already exists in the assignment map.
			log.Printf("Warning: Newly created project %d (NsId: %s) was unexpectedly found in the assignment map.", currentProjectID, infraNs.ID)
		} else {
			// Existing project is already assigned to at least one workspace
			log.Printf("Project %d (NsId: %s) is already assigned to a workspace. Skipping default assignment.", currentProjectID, infraNs.ID)
		}
	}

	log.Printf("Project synchronization finished. Added %d new projects. Assigned %d existing unassigned projects to default workspace.", addedCount, assignedToDefaultCount)
	return nil
}

// CreateProject 새로운 프로젝트를 생성하고 기본 워크스페이스에 할당합니다.
func (s *ProjectService) CreateProject(project *model.Project) error {
	// 프로젝트 생성
	if err := s.projectRepo.CreateProject(project); err != nil {
		return err
	}

	// 기본 워크스페이스 조회
	defaultWorkspace, err := s.workspaceRepo.FindWorkspaceByName(os.Getenv("DEFAULT_WORKSPACE_NAME"))
	if err != nil {
		return fmt.Errorf("기본 워크스페이스를 찾을 수 없습니다: %v", err)
	}

	// 프로젝트를 기본 워크스페이스에 할당
	if err := s.projectRepo.AddProjectWorkspaceAssociation(defaultWorkspace.ID, project.ID); err != nil {
		// 프로젝트 생성은 성공했지만 워크스페이스 할당에 실패한 경우
		// 프로젝트를 삭제하고 에러 반환
		s.projectRepo.DeleteProject(project.ID)
		return fmt.Errorf("프로젝트를 기본 워크스페이스에 할당하는데 실패했습니다: %v", err)
	}

	return nil
}
