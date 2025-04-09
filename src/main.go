package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/handler"
	"github.com/m-cmp/mc-iam-manager/middleware"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title MC IAM Manager API
// @version 1.0
// @description MC IAM Manager API Documentation
// @host localhost:8082
// @BasePath /api/v1
func main() {
	// .env 파일 로드 (프로젝트 루트에서 찾도록 수정)
	envPath := filepath.Join("..", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: .env 파일을 로드하는데 실패했습니다: %v", err)
	}

	// // 데이터베이스 초기화
	// db, err := config.InitDB()
	// if err != nil {
	// 	log.Fatalf("Failed to initialize database: %v", err)
	// }
	// 데이터베이스 초기화
	dbConfig := config.NewDatabaseConfig()
	db, err := gorm.Open(postgres.Open(dbConfig.GetDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Keycloak 초기화
	if err := config.InitKeycloak(); err != nil {
		log.Fatalf("Failed to initialize Keycloak: %v", err)
	}

	// Echo 인스턴스 생성
	e := echo.New()

	// 미들웨어 설정
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

	// Repository 초기화
	userRepo := repository.NewUserRepository(nil, config.KC, config.KC.Client)

	platformRoleRepo := repository.NewPlatformRoleRepository(db)
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db)

	// Service 초기화
	userService := service.NewUserService(userRepo, config.KC, config.KC.Client)
	platformRoleService := service.NewPlatformRoleService(platformRoleRepo)
	workspaceRoleService := service.NewWorkspaceRoleService(workspaceRoleRepo)

	// 핸들러 초기화
	authHandler := handler.NewAuthHandler(config.KC)
	platformRoleHandler := handler.NewPlatformRoleHandler(platformRoleService)
	workspaceRoleHandler := handler.NewWorkspaceRoleHandler(workspaceRoleService)
	userHandler := handler.NewUserHandler(userService)

	// 라우트 설정
	e.GET("/readyz", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// // 인증 라우트
	// e.POST("/login", authHandler.Login)
	// e.POST("/logout", authHandler.Logout)
	// e.POST("/refresh", authHandler.RefreshToken)

	api := e.Group("/api")
	// 인증 라우트
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/logout", authHandler.Logout)
	api.POST("/auth/refresh", authHandler.RefreshToken)

	// 사용자 라우트
	api.Use(middleware.KeycloakAuthMiddleware)
	{
		api.GET("/users", userHandler.GetUsers)
		api.GET("/users/:id", userHandler.GetUserByID)
		api.GET("/users/username/:username", userHandler.GetUserByUsername)
		api.POST("/users", userHandler.CreateUser)
		api.PUT("/users/:id", userHandler.UpdateUser)
		api.DELETE("/users/:id", userHandler.DeleteUser)
	}

	// 플랫폼 역할 라우트
	api.GET("/platform-roles", platformRoleHandler.List)
	api.GET("/platform-roles/:id", platformRoleHandler.GetByID)
	api.POST("/platform-roles", platformRoleHandler.Create)
	api.PUT("/platform-roles/:id", platformRoleHandler.Update)
	api.DELETE("/platform-roles/:id", platformRoleHandler.Delete)

	// 워크스페이스 역할 라우트
	api.GET("/workspace-roles", workspaceRoleHandler.List)
	api.GET("/workspace-roles/:id", workspaceRoleHandler.GetByID)
	api.POST("/workspace-roles", workspaceRoleHandler.Create)
	api.PUT("/workspace-roles/:id", workspaceRoleHandler.Update)
	api.DELETE("/workspace-roles/:id", workspaceRoleHandler.Delete)

	// ... existing code ...
	// 권한 관리 API 라우트
	permissionHandler := handler.NewPermissionHandler(service.NewPermissionService(repository.NewPermissionRepository(db)))
	api.POST("/permissions", permissionHandler.Create)
	api.GET("/permissions", permissionHandler.List)
	api.GET("/permissions/:id", permissionHandler.GetByID)
	api.PUT("/permissions/:id", permissionHandler.Update)
	api.DELETE("/permissions/:id", permissionHandler.Delete)
	api.POST("/roles/:roleId/permissions/:permissionId", permissionHandler.AssignRolePermission)
	api.DELETE("/roles/:roleId/permissions/:permissionId", permissionHandler.RemoveRolePermission)
	api.GET("/roles/:roleId/permissions", permissionHandler.GetRolePermissions)

	// 서버 시작
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
