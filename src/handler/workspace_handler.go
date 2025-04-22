package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Ensure gorm is imported
)

// WorkspaceHandler 워크스페이스 관리 핸들러
type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
	// db *gorm.DB // Not needed directly in handler
}

// NewWorkspaceHandler 새 WorkspaceHandler 인스턴스 생성
func NewWorkspaceHandler(db *gorm.DB) *WorkspaceHandler { // Accept db, remove service param
	// Initialize service internally
	workspaceService := service.NewWorkspaceService(db)
	return &WorkspaceHandler{workspaceService: workspaceService}
}

// CreateWorkspace godoc
// @Summary 워크스페이스 생성
// @Description 새로운 워크스페이스를 생성합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body model.Workspace true "워크스페이스 정보 (ID, CreatedAt, UpdatedAt, Projects 제외)"
// @Success 201 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces [post]
func (h *WorkspaceHandler) CreateWorkspace(c echo.Context) error {
	var workspace model.Workspace
	if err := c.Bind(&workspace); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// Ensure ID is not set by client
	workspace.ID = 0

	if err := h.workspaceService.Create(&workspace); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 생성 실패: %v", err)})
	}
	// Return the created workspace, including the generated ID
	return c.JSON(http.StatusCreated, workspace)
}

// ListWorkspaces godoc
// @Summary 모든 워크스페이스 조회
// @Description 모든 워크스페이스 목록을 조회합니다 (연결된 프로젝트 정보 포함).
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces [get]
func (h *WorkspaceHandler) ListWorkspaces(c echo.Context) error {
	workspaces, err := h.workspaceService.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 목록 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspaces)
}

// GetWorkspaceByID godoc
// @Summary ID로 워크스페이스 조회
// @Description ID로 특정 워크스페이스를 조회합니다 (연결된 프로젝트 정보 포함).
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 200 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id} [get]
func (h *WorkspaceHandler) GetWorkspaceByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	workspace, err := h.workspaceService.GetByID(uint(id))
	if err != nil {
		// Check for specific "not found" error
		if err.Error() == "workspace not found" { // Assuming service returns this error string
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// GetWorkspaceByName godoc
// @Summary 이름으로 워크스페이스 조회
// @Description 이름으로 특정 워크스페이스를 조회합니다 (연결된 프로젝트 정보 포함).
// @Tags workspaces
// @Accept json
// @Produce json
// @Param name path string true "워크스페이스 이름"
// @Success 200 {object} model.Workspace
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/name/{name} [get]
func (h *WorkspaceHandler) GetWorkspaceByName(c echo.Context) error {
	name := c.Param("name")

	workspace, err := h.workspaceService.GetByName(name)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// UpdateWorkspace godoc
// @Summary 워크스페이스 수정
// @Description 기존 워크스페이스 정보를 부분적으로 수정합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param updates body object true "수정할 필드와 값 (예: {\"name\": \"New Name\", \"description\": \"New Desc\"})"
// @Success 200 {object} model.Workspace "업데이트된 워크스페이스 정보"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식 또는 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id} [put]
func (h *WorkspaceHandler) UpdateWorkspace(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("잘못된 요청 형식입니다: %v", err)})
	}

	// Prevent updating ID or CreatedAt/UpdatedAt via map
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "projects") // Prevent direct update of associations

	if len(updates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "업데이트할 필드가 없습니다"})
	}

	if err := h.workspaceService.Update(uint(id), updates); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 업데이트 실패: %v", err)})
	}

	// Return updated workspace
	updatedWorkspace, err := h.workspaceService.GetByID(uint(id))
	if err != nil {
		// Log error but return success as update itself was successful
		fmt.Printf("Warning: Failed to fetch updated workspace (id: %d): %v\n", id, err)
		return c.JSON(http.StatusOK, updates) // Return updates map as confirmation
	}
	return c.JSON(http.StatusOK, updatedWorkspace)
}

// DeleteWorkspace godoc
// @Summary 워크스페이스 삭제
// @Description 워크스페이스를 삭제합니다. 연결된 프로젝트와의 관계도 해제됩니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id} [delete]
func (h *WorkspaceHandler) DeleteWorkspace(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	if err := h.workspaceService.Delete(uint(id)); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 삭제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// ListUsersAndRolesByWorkspace godoc
// @Summary 워크스페이스 사용자 및 역할 목록 조회
// @Description 특정 워크스페이스에 속한 모든 사용자와 각 사용자의 역할을 조회합니다.
// @Tags workspaces, users, roles
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 200 {array} service.UserWithRoles "성공 시 사용자 및 역할 목록 반환"
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/users [get]
func (h *WorkspaceHandler) ListUsersAndRolesByWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	usersWithRoles, err := h.workspaceService.GetUsersAndRolesByWorkspaceID(uint(workspaceID))
	if err != nil {
		// Handle not found error from service (which checks workspace existence)
		if errors.Is(err, repository.ErrWorkspaceNotFound) { // Assuming service passes this error up
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 및 역할 목록 조회 실패: %v", err)})
	}

	// Return empty list if no users are associated
	if usersWithRoles == nil {
		usersWithRoles = []service.UserWithRoles{} // Ensure we return [] instead of null
	}

	return c.JSON(http.StatusOK, usersWithRoles)
}

// ListProjectsByWorkspace godoc
// @Summary 워크스페이스에 연결된 프로젝트 목록 조회
// @Description 특정 워크스페이스 ID에 연결된 모든 프로젝트 목록을 조회합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 200 {array} model.Project "성공 시 프로젝트 목록 반환"
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/projects [get]
func (h *WorkspaceHandler) ListProjectsByWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	projects, err := h.workspaceService.GetProjectsByWorkspaceID(uint(workspaceID))
	if err != nil {
		// Handle not found error from service (which checks workspace existence)
		if err.Error() == "workspace not found" { // Assuming service returns this specific error
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 목록 조회 실패: %v", err)})
	}

	// Return empty list if no projects are associated, instead of 404
	if projects == nil {
		projects = []model.Project{} // Ensure we return [] instead of null
	}

	return c.JSON(http.StatusOK, projects)
}

// AddProjectToWorkspace godoc
// @Summary 워크스페이스에 프로젝트 연결
// @Description 특정 워크스페이스에 프로젝트를 연결합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param projectId path int true "프로젝트 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 워크스페이스 또는 프로젝트를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/projects/{projectId} [post]
func (h *WorkspaceHandler) AddProjectToWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	if err := h.workspaceService.AddProjectToWorkspace(uint(workspaceID), uint(projectID)); err != nil {
		// Handle not found errors from service
		if err.Error() == "workspace not found" || err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		// Handle potential duplicate errors if repo doesn't ignore them
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 연결 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveProjectFromWorkspace godoc
// @Summary 워크스페이스에서 프로젝트 연결 해제
// @Description 특정 워크스페이스에서 프로젝트 연결을 해제합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param projectId path int true "프로젝트 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/projects/{projectId} [delete]
func (h *WorkspaceHandler) RemoveProjectFromWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	if err := h.workspaceService.RemoveProjectFromWorkspace(uint(workspaceID), uint(projectID)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 연결 해제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}
