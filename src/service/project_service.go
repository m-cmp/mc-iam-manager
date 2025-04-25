package service

import (
	"context"       // Add context import
	"encoding/json" // Add json import for request body
	"errors"

	// "errors"        // Remove unused import
	"fmt" // Add fmt import for errors
	"log" // Add log import
	"os"  // Import os package to read environment variables

	// "net/http"      // Remove unused import

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi" // Import mcmpapi model
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm" // Ensure gorm is imported
)

// ProjectService 프로젝트 관리 서비스
type ProjectService struct {
	projectRepo    *repository.ProjectRepository
	workspaceRepo  *repository.WorkspaceRepository // Needed for association checks
	mcmpApiService McmpApiService                  // Added dependency back
	db             *gorm.DB                        // Add DB field
}

// NewProjectService 새 ProjectService 인스턴스 생성
func NewProjectService(db *gorm.DB, mcmpApiService McmpApiService) *ProjectService { // Accept db and mcmpApiService
	// Initialize repositories internally
	projectRepo := repository.NewProjectRepository(db)
	workspaceRepo := repository.NewWorkspaceRepository(db)
	return &ProjectService{
		db:             db, // Store db
		projectRepo:    projectRepo,
		workspaceRepo:  workspaceRepo,
		mcmpApiService: mcmpApiService, // Initialize dependency
	}
}

// Create 프로젝트 생성 (mc-infra-manager 호출 및 DB 저장)
func (s *ProjectService) Create(ctx context.Context, project *model.Project) error { // Added context parameter back
	log.Printf("Attempting to create namespace in mc-infra-manager for project: %s", project.Name)

	// 1. Call mc-infra-manager PostNs API
	nsRequestBody := map[string]string{
		"name":        project.Name,
		"description": project.Description,
	}
	// bodyBytes, err := json.Marshal(nsRequestBody) // Don't marshal here
	// if err != nil {
	// 	log.Printf("Error marshalling request body for PostNs: %v", err)
	// 	return fmt.Errorf("failed to marshal request body for PostNs: %w", err)
	// }

	callReq := &mcmpapi.McmpApiCallRequest{
		ServiceName: "mc-infra-manager",
		ActionName:  "Postns", // Corrected action name based on previous analysis
		RequestParams: mcmpapi.McmpApiRequestParams{
			Body: nsRequestBody, // Pass the original map directly
		},
	}

	statusCode, respBody, serviceVersion, calledURL, err := s.mcmpApiService.McmpApiCall(ctx, callReq) // Get new return values
	if err != nil {
		// Include version and URL in the error message
		log.Printf("Error calling %s(v%s) %s (URL: %s): %v (status code: %d)", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, err, statusCode)
		return fmt.Errorf("failed to call %s(v%s) %s (URL: %s): %w (status code: %d)", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, err, statusCode)
	}
	if statusCode < 200 || statusCode >= 300 {
		// Include version and URL in the error message
		log.Printf("%s(v%s) %s call failed (URL: %s): status code %d, response: %s", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, statusCode, string(respBody))
		var errorResp map[string]interface{}
		errMsg := fmt.Sprintf("%s(v%s) %s call failed with status code %d (URL: %s)", callReq.ServiceName, serviceVersion, callReq.ActionName, statusCode, calledURL) // Base error message
		if json.Unmarshal(respBody, &errorResp) == nil {
			if msg, ok := errorResp["message"].(string); ok {
				errMsg = fmt.Sprintf("%s(v%s) error: %s (URL: %s, Status: %d)", callReq.ServiceName, serviceVersion, msg, calledURL, statusCode) // More specific message if possible
			}
		}
		return errors.New(errMsg) // Return as a simple error for now, or wrap if needed
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

	// 2. Create project in local DB
	if err := s.projectRepo.Create(project); err != nil {
		return err // Return DB creation error
	}

	// 3. Assign to default workspace
	defaultWsName := os.Getenv("DEFAULT_WORKSPACE_NAME")
	if defaultWsName == "" {
		defaultWsName = "default" // Fallback to "default" if env var is not set
	}
	log.Printf("Assigning newly created project (ID: %d) to default workspace '%s'", project.ID, defaultWsName)
	defaultWs, err := s.workspaceRepo.GetByName(defaultWsName)
	if err != nil {
		log.Printf("Error finding default workspace '%s': %v. Skipping assignment.", defaultWsName, err)
		// Decide if this should be a critical error or just a warning
		// For now, log a warning and return success as the project itself was created.
		return nil // Or return fmt.Errorf("failed to find default workspace: %w", err)
	}
	if err := s.projectRepo.AddWorkspaceAssociation(project.ID, defaultWs.ID); err != nil {
		log.Printf("Error assigning project %d to default workspace %d: %v", project.ID, defaultWs.ID, err)
		// Log a warning, but the project creation was successful.
		return nil // Or return fmt.Errorf("failed to assign project to default workspace: %w", err)
	}
	log.Printf("Successfully assigned project %d to default workspace %d", project.ID, defaultWs.ID)

	return nil // Project created and assigned (or assignment failed but logged)
}

// List 모든 프로젝트 조회
func (s *ProjectService) List() ([]model.Project, error) {
	return s.projectRepo.List()
}

// GetByID ID로 프로젝트 조회
func (s *ProjectService) GetByID(id uint) (*model.Project, error) {
	return s.projectRepo.GetByID(id)
}

// GetByName 이름으로 프로젝트 조회
func (s *ProjectService) GetByName(name string) (*model.Project, error) {
	return s.projectRepo.GetByName(name)
}

// Update 프로젝트 정보 부분 업데이트
func (s *ProjectService) Update(id uint, updates map[string]interface{}) error {
	_, err := s.projectRepo.GetByID(id)
	if err != nil {
		// Propagate the error (e.g., ErrProjectNotFound)
		return err
	}
	return s.projectRepo.Update(id, updates)
}

// Delete 프로젝트 삭제
func (s *ProjectService) Delete(id uint) error {
	// Check if project exists before deleting
	_, err := s.projectRepo.GetByID(id)
	if err != nil {
		return err // Return error if not found or other DB error
	}
	// TODO: Consider adding logic here or in handler to call mc-infra-manager DeleteNs API
	return s.projectRepo.Delete(id)
}

// AddWorkspaceToProject 프로젝트에 워크스페이스 연결
func (s *ProjectService) AddWorkspaceToProject(projectID, workspaceID uint) error {
	_, errPr := s.projectRepo.GetByID(projectID)
	if errPr != nil {
		return errPr
	}
	_, errWs := s.workspaceRepo.GetByID(workspaceID)
	if errWs != nil {
		return errWs
	}
	return s.projectRepo.AddWorkspaceAssociation(projectID, workspaceID)
}

// RemoveWorkspaceFromProject 프로젝트에서 워크스페이스 연결 제거
func (s *ProjectService) RemoveWorkspaceFromProject(projectID, workspaceID uint) error {
	// Optional: Check if project and workspace exist before attempting removal
	return s.projectRepo.RemoveWorkspaceAssociation(projectID, workspaceID)
}

// SyncProjectsWithInfraManager mc-infra-manager의 네임스페이스와 로컬 프로젝트 동기화
func (s *ProjectService) SyncProjectsWithInfraManager(ctx context.Context) error {
	log.Println("Starting project synchronization with mc-infra-manager...")

	// 1. Call mc-infra-manager GetAllNs API
	callReq := &mcmpapi.McmpApiCallRequest{
		ServiceName: "mc-infra-manager",
		ActionName:  "GetAllNs",
		RequestParams: mcmpapi.McmpApiRequestParams{ // No params needed for GetAllNs
			PathParams:  nil,
			QueryParams: nil,
			Body:        nil,
		},
	}

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
	localProjects, err := s.projectRepo.List()
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
	assignedProjectMap, err := s.projectRepo.GetAllProjectWorkspaceAssignments()
	if err != nil {
		log.Printf("Error getting project workspace assignments: %v", err)
		return fmt.Errorf("failed to get project assignments: %w", err)
	}
	log.Printf("Found %d projects assigned to at least one workspace.", len(assignedProjectMap))

	// Get default workspace ID once
	defaultWsName := os.Getenv("DEFAULT_WORKSPACE_NAME")
	if defaultWsName == "" {
		defaultWsName = "default" // Fallback
	}
	defaultWs, err := s.workspaceRepo.GetByName(defaultWsName)
	if err != nil {
		log.Printf("Error finding default workspace '%s': %v. Cannot assign projects.", defaultWsName, err)
		// If default workspace doesn't exist, we can't proceed with assignment.
		// Depending on requirements, we might return an error or just log and continue.
		// Let's return an error for now, as assignment is a key part of this logic.
		return fmt.Errorf("failed to find default workspace: %w", err)
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
				existingProject = &p
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
			if err := s.projectRepo.Create(newProject); err != nil {
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
			if assignErr := s.projectRepo.AddWorkspaceAssociation(currentProjectID, defaultWs.ID); assignErr != nil {
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
