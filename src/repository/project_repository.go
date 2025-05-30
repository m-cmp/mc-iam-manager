package repository

import (
	"errors"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	// "github.com/m-cmp/mc-iam-manager/service" // Remove service import
	"gorm.io/gorm"
)

var (
	ErrProjectNotFound = errors.New("project not found")
)

// ProjectRepository 프로젝트 데이터 관리
type ProjectRepository struct {
	db *gorm.DB
	// mcmpApiService service.McmpApiService // Removed dependency
}

// NewProjectRepository 새 ProjectRepository 인스턴스 생성
func NewProjectRepository(db *gorm.DB) *ProjectRepository { // Removed parameter
	return &ProjectRepository{
		db: db,
		// mcmpApiService: mcmpApiService, // Removed initialization
	}
}

// Create 프로젝트 생성
func (r *ProjectRepository) CreateProject(project *model.Project) error {
	query := r.db.Create(project)
	if err := query.Error; err != nil {
		return err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Create SQL Query: %s", sql)
	log.Printf("Create SQL Args: %v", args)
	log.Printf("Create Created ID: %d", project.ID)

	return nil
}

// List 모든 프로젝트 조회 (워크스페이스 정보 포함)
func (r *ProjectRepository) FindProjects() ([]*model.Project, error) {
	var projects []*model.Project
	query := r.db.Preload("Workspaces").Find(&projects)
	if err := query.Error; err != nil {
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("List SQL Query: %s", sql)
	log.Printf("List SQL Args: %v", args)
	log.Printf("List Result Count: %d", len(projects))

	return projects, nil
}

// GetByID ID로 프로젝트 조회
func (r *ProjectRepository) FindProjectByProjectID(id uint) (*model.Project, error) {
	var project model.Project
	query := r.db.First(&project, "id = ?", id)
	if err := query.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetByID SQL Query: %s", sql)
	log.Printf("GetByID SQL Args: %v", args)

	return &project, nil
}

// GetByName 이름으로 프로젝트 조회 (워크스페이스 정보 포함)
func (r *ProjectRepository) FindProjectByProjectName(name string) (*model.Project, error) {
	var project model.Project
	query := r.db.Preload("Workspaces").Where("name = ?", name).First(&project)
	if err := query.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetByName SQL Query: %s", sql)
	log.Printf("GetByName SQL Args: %v", args)

	return &project, nil
}

// Update 프로젝트 정보 업데이트
func (r *ProjectRepository) UpdateProject(id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	result := r.db.Model(&model.Project{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// GetAllProjectWorkspaceAssignments 모든 프로젝트-워크스페이스 할당 정보 조회
// 프로젝트 ID를 키로 하고, 할당 여부(true)를 값으로 하는 맵 반환
func (r *ProjectRepository) FindAllProjectWorkspaceAssignments() (map[uint]bool, error) {
	var assignments []struct {
		ProjectID uint `gorm:"column:project_id"`
	}
	// Select distinct project_id from the join table
	if err := r.db.Table("mcmp_workspace_projects").Distinct("project_id").Find(&assignments).Error; err != nil {
		return nil, err
	}

	assignmentMap := make(map[uint]bool)
	for _, assign := range assignments {
		assignmentMap[assign.ProjectID] = true
	}
	return assignmentMap, nil
}

// Delete 프로젝트 삭제
func (r *ProjectRepository) DeleteProject(id uint) error {
	query := r.db.Delete(&model.Project{}, id)
	if err := query.Error; err != nil {
		return err
	}
	if query.RowsAffected == 0 {
		return ErrProjectNotFound
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("Delete SQL Query: %s", sql)
	log.Printf("Delete SQL Args: %v", args)
	log.Printf("Delete Affected Rows: %d", query.RowsAffected)

	return nil
}

// AddWorkspaceAssociation 프로젝트에 워크스페이스 연결 추가
func (r *ProjectRepository) AddProjectWorkspaceAssociation(projectID, workspaceID uint) error {
	var project model.Project
	project.ID = projectID
	var workspace model.Workspace
	workspace.ID = workspaceID

	err := r.db.Model(&project).Association("Workspaces").Append(&workspace)
	if err != nil {
		return err
	}

	// SQL 쿼리 로깅
	log.Printf("AddWorkspaceAssociation: Project ID %d, Workspace ID %d", projectID, workspaceID)

	return nil
}

// RemoveWorkspaceAssociation 프로젝트에서 워크스페이스 연결 제거
func (r *ProjectRepository) RemoveProjectWorkspaceAssociation(projectID, workspaceID uint) error {
	var project model.Project
	project.ID = projectID
	var workspace model.Workspace
	workspace.ID = workspaceID

	err := r.db.Model(&project).Association("Workspaces").Delete(&workspace)
	if err != nil {
		return err
	}

	// SQL 쿼리 로깅
	log.Printf("RemoveWorkspaceAssociation: Project ID %d, Workspace ID %d", projectID, workspaceID)

	return nil
}

// GetAssignedWorkspaces 프로젝트에 할당된 워크스페이스 목록을 조회합니다.
func (r *ProjectRepository) FindAssignedWorkspaces(projectID uint) ([]*model.Workspace, error) {
	var workspaces []*model.Workspace
	err := r.db.Model(&model.Project{ID: projectID}).
		Association("Workspaces").
		Find(&workspaces)
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}
