package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
)

// ProjectHandler 프로젝트 관리 핸들러
type ProjectHandler struct {
	projectService *service.ProjectService
}

// NewProjectHandler 새 ProjectHandler 인스턴스 생성
func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

// CreateProject godoc
// @Summary 프로젝트 생성
// @Description 새로운 프로젝트를 생성합니다.
// @Tags projects
// @Accept json
// @Produce json
// @Param project body model.Project true "프로젝트 정보 (ID, CreatedAt, UpdatedAt, Workspaces 제외)"
// @Success 201 {object} model.Project
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	var project model.Project
	if err := c.Bind(&project); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	project.ID = 0 // Ensure ID is not set by client

	if err := h.projectService.Create(&project); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 생성 실패: %v", err)})
	}
	return c.JSON(http.StatusCreated, project)
}

// ListProjects godoc
// @Summary 모든 프로젝트 조회
// @Description 모든 프로젝트 목록을 조회합니다 (연결된 워크스페이스 정보 포함).
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {array} model.Project
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /projects [get]
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	projects, err := h.projectService.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 목록 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, projects)
}

// GetProjectByID godoc
// @Summary ID로 프로젝트 조회
// @Description ID로 특정 프로젝트를 조회합니다 (연결된 워크스페이스 정보 포함).
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "프로젝트 ID"
// @Success 200 {object} model.Project
// @Failure 400 {object} map[string]string "error: 잘못된 프로젝트 ID"
// @Failure 404 {object} map[string]string "error: 프로젝트를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /projects/{id} [get]
func (h *ProjectHandler) GetProjectByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	project, err := h.projectService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, project)
}

// UpdateProject godoc
// @Summary 프로젝트 수정
// @Description 기존 프로젝트 정보를 부분적으로 수정합니다.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "프로젝트 ID"
// @Param updates body object true "수정할 필드와 값 (예: {\"name\": \"New Name\", \"description\": \"New Desc\", \"nsid\": \"new-ns\"})"
// @Success 200 {object} model.Project "업데이트된 프로젝트 정보"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식 또는 ID"
// @Failure 404 {object} map[string]string "error: 프로젝트를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("잘못된 요청 형식입니다: %v", err)})
	}

	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "workspaces")

	if len(updates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "업데이트할 필드가 없습니다"})
	}

	if err := h.projectService.Update(uint(id), updates); err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 업데이트 실패: %v", err)})
	}

	updatedProject, err := h.projectService.GetByID(uint(id))
	if err != nil {
		fmt.Printf("Warning: Failed to fetch updated project (id: %d): %v\n", id, err)
		return c.JSON(http.StatusOK, updates)
	}
	return c.JSON(http.StatusOK, updatedProject)
}

// DeleteProject godoc
// @Summary 프로젝트 삭제
// @Description 프로젝트를 삭제합니다. 연결된 워크스페이스와의 관계도 해제됩니다.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "프로젝트 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 프로젝트 ID"
// @Failure 404 {object} map[string]string "error: 프로젝트를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	if err := h.projectService.Delete(uint(id)); err != nil {
		if err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 삭제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// AddWorkspaceToProject godoc
// @Summary 프로젝트에 워크스페이스 연결
// @Description 특정 프로젝트에 워크스페이스를 연결합니다.
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
// @Router /projects/{id}/workspaces/{workspaceId} [post]
func (h *ProjectHandler) AddWorkspaceToProject(c echo.Context) error {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	if err := h.projectService.AddWorkspaceToProject(uint(projectID), uint(workspaceID)); err != nil {
		if err.Error() == "project not found" || err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 연결 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveWorkspaceFromProject godoc
// @Summary 프로젝트에서 워크스페이스 연결 해제
// @Description 특정 프로젝트에서 워크스페이스 연결을 해제합니다.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "프로젝트 ID"
// @Param workspaceId path int true "워크스페이스 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /projects/{id}/workspaces/{workspaceId} [delete]
func (h *ProjectHandler) RemoveWorkspaceFromProject(c echo.Context) error {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	if err := h.projectService.RemoveWorkspaceFromProject(uint(projectID), uint(workspaceID)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 연결 해제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}
