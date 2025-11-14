package handler

import (
	"fmt"
	"net/http"

	// Add json import
	"log" // Add log import

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	// Removed duplicate echo import
	"github.com/m-cmp/mc-iam-manager/model" // Import mcmpapi model for request
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
)

// ProjectHandler 프로젝트 관련 HTTP 요청을 처리하는 핸들러
type ProjectHandler struct {
	projectService   *service.ProjectService
	workspaceService *service.WorkspaceService // WorkspaceService 추가
	userService      *service.UserService
	permissionRepo   *repository.MciamPermissionRepository
	mcmpApiService   service.McmpApiService
	db               *gorm.DB
}

// NewProjectHandler 새로운 ProjectHandler 인스턴스 생성
func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{
		projectService:   service.NewProjectService(db),
		workspaceService: service.NewWorkspaceService(db), // WorkspaceService 초기화
		userService:      service.NewUserService(db),
		permissionRepo:   repository.NewMciamPermissionRepository(db),
		mcmpApiService:   service.NewMcmpApiService(db),
		db:               db,
	}
}

// CreateProject godoc
// @Summary Create new project
// @Description Create a new project with the specified information. Optionally specify a workspace to assign the project to.
// @Tags projects
// @Accept json
// @Produce json
// @Param project body model.CreateProjectRequest true "Project Info"
// @Success 201 {object} model.Project
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/projects [post]
// @Id createProject
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	var req model.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	project := &model.Project{
		Name:        req.Name,
		Description: req.Description,
	}

	// Parse optional workspaceId
	var workspaceID uint
	if req.WorkspaceID != "" {
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt
	}

	// Call the service Create method with optional workspaceID
	if err := h.projectService.Create(c.Request().Context(), project, workspaceID); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "워크스페이스를 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 생성 실패: %v", err)})
	}

	log.Printf("Successfully created project '%s' (ID: %d, NsId: %s)", project.Name, project.ID, project.NsId)
	return c.JSON(http.StatusCreated, project)
}

// ListProjects godoc
// @Summary List all projects
// @Description Retrieve a list of all projects.
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {array} model.Project
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/projects/list [post]
// @Id listProjects
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	var req model.ProjectFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	var projects []*model.Project
	// if hasListAllPermission {
	// User has permission to list all workspaces
	projects, err := h.projectService.ListProjects(&req)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 목록 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, projects)
}

// GetProjectByID godoc
// @Summary Get project by ID
// @Description Retrieve project details by project ID.
// @Tags projects
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 200 {object} model.Project
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/projects/{id} [get]
// @Id getProjectByID
func (h *ProjectHandler) GetProjectByID(c echo.Context) error {
	// Parse DB ID (uint) from path parameter
	projectIDInt, err := util.StringToUint(c.Param("projectId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
	}

	project, err := h.projectService.GetProjectByID(projectIDInt)
	if err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, project)
}

// GetProjectByName godoc
// @Summary Get project by name
// @Description Get project details by name
// @Tags projects
// @Accept json
// @Produce json
// @Param name path string true "Project Name"
// @Success 200 {object} model.Project
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/projects/name/{projectName} [get]
// @Id getProjectByName
func (h *ProjectHandler) GetProjectByName(c echo.Context) error {
	name := c.Param("projectName")

	project, err := h.projectService.GetProjectByName(name)
	if err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, project)
}

// UpdateProject godoc
// @Summary Update project
// @Description Update the details of an existing project.
// @Tags projects
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Param project body model.Project true "Project Info"
// @Success 200 {object} model.Project
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/projects/{id} [put]
// @Id updateProject
func (h *ProjectHandler) UpdateProject(c echo.Context) error {

	var project model.Project
	if err := c.Bind(&project); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	projectIDInt, err := util.StringToUint(c.Param("projectId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
	}

	updates := map[string]interface{}{
		"name":        project.Name,
		"description": project.Description,
	}

	if err := h.projectService.UpdateProject(projectIDInt, updates); err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 수정 실패: %v", err)})
	}

	// Get updated project
	updatedProject, err := h.projectService.GetProjectByID(projectIDInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("수정된 프로젝트 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, updatedProject)
}

// DeleteProject godoc
// @Summary Delete project
// @Description Delete a project by its ID.
// @Tags projects
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/projects/{id} [delete]
// @Id deleteProject
func (h *ProjectHandler) DeleteProject(c echo.Context) error {

	projectIDInt, err := util.StringToUint(c.Param("projectId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
	}

	if err := h.projectService.DeleteProject(projectIDInt); err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 삭제 실패: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetProjectWorkspaces godoc
// @Summary Get workspaces assigned to project
// @Description Retrieve list of workspaces that the project is assigned to
// @Tags projects
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 200 {array} model.Workspace
// @Failure 400 {object} map[string]string "error: Invalid project ID"
// @Failure 404 {object} map[string]string "error: Project not found"
// @Failure 500 {object} map[string]string "error: Internal server error"
// @Security BearerAuth
// @Router /api/projects/id/{projectId}/workspaces [get]
// @Id getProjectWorkspaces
func (h *ProjectHandler) GetProjectWorkspaces(c echo.Context) error {
	projectIDInt, err := util.StringToUint(c.Param("projectId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	workspaces, err := h.projectService.GetProjectWorkspaces(projectIDInt)
	if err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Project not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to retrieve workspace list: %v", err),
		})
	}

	// 빈 배열 처리
	if workspaces == nil {
		workspaces = []*model.Workspace{}
	}

	return c.JSON(http.StatusOK, workspaces)
}

// SyncProjects godoc
// @Summary mc-infra-manager와 프로젝트 동기화
// @Description mc-infra-manager의 네임스페이스 목록을 조회하여 로컬 DB에 없는 프로젝트를 추가합니다.
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Project synchronization successful"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류 또는 동기화 실패"
// @Security BearerAuth
// @Router /api/setup/sync-projects [post]
// @Id syncProjects
func (h *ProjectHandler) SyncProjects(c echo.Context) error {
	log.Println("Received request to sync projects with mc-infra-manager")
	if err := h.projectService.SyncProjectsWithInfraManager(c.Request().Context()); err != nil {
		log.Printf("Error during project synchronization: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 동기화 실패: %v", err)})
	}
	log.Println("Project synchronization completed successfully")
	return c.JSON(http.StatusOK, map[string]string{"message": "Project synchronization successful"})
}

// AddWorkspaceToProject godoc
// @Summary 프로젝트에 워크스페이스 연결
// @Description 프로젝트에 워크스페이스를 연결합니다.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "프로젝트 ID"
// @Param workspaceId path int true "워크스페이스 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 프로젝트 또는 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/projects/{id}/workspaces/{workspaceId} [post]
// @Id addWorkspaceToProject
func (h *ProjectHandler) AddWorkspaceToProject(c echo.Context) error {
	var req model.WorkspaceProjectMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	if req.ProjectIDs == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	for _, projectID := range req.ProjectIDs {
		projectIDInt, err := util.StringToUint(projectID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
		}

		if err := h.workspaceService.AddProjectToWorkspace(workspaceIDInt, projectIDInt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 연결 실패: %v", err)})
		}
		// if err := h.projectService.AddWorkspaceAssociation(projectIDInt, workspaceIDInt); err != nil { // 프로젝트 ID는 배열이므로 반복문으로 처리
		// if err.Error() == "project not found" || err.Error() == "workspace not found" {
		// 	return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		// }
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 연결 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveWorkspaceFromProject 프로젝트에서 워크스페이스 연결 해제
// @Summary Remove workspace from project
// @Description Remove a workspace from a project
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Id removeWorkspaceFromProject
func (h *ProjectHandler) RemoveWorkspaceFromProject(c echo.Context) error {
	var req model.WorkspaceProjectMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	if req.ProjectIDs == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	for _, projectID := range req.ProjectIDs {
		projectIDInt, err := util.StringToUint(projectID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
		}

		if err := h.workspaceService.RemoveProjectFromWorkspace(workspaceIDInt, projectIDInt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.NoContent(http.StatusNoContent)
}
