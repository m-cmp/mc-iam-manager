package service

import (
	"context"       // Add context import
	"encoding/json" // Add json import for request body
	"errors"

	// "errors"        // Remove unused import
	"fmt" // Add fmt import for errors
	"log" // Add log import

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

	// Only call the repository to create the project in the DB
	return s.projectRepo.Create(project)
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
