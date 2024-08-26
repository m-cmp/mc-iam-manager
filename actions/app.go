package actions

import (
	"net/http"
	"sync"

	"github.com/m-cmp/mc-iam-manager/middleware"
	"github.com/m-cmp/mc-iam-manager/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v3/pop/popmw"
	"github.com/gobuffalo/envy"
	contenttype "github.com/gobuffalo/mw-contenttype"
	i18n "github.com/gobuffalo/mw-i18n/v2"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"
)

var ENV = envy.Get("GO_ENV", "development")

var (
	app     *buffalo.App
	appOnce sync.Once
	T       *i18n.Translator
)

func App() *buffalo.App {
	appOnce.Do(func() {
		app = buffalo.New(buffalo.Options{
			Env:          ENV,
			SessionStore: sessions.Null{},
			PreWares: []buffalo.PreWare{
				cors.Default().Handler,
			},
			SessionName: "_github.com/m-cmp/mc-iam-manager_session",
		})

		app.Use(paramlogger.ParameterLogger)
		app.Use(contenttype.Set("application/json"))
		app.Use(popmw.Transaction(models.DB))
		app.Use(middleware.IsAuthMiddleware)
		app.Use(middleware.SetContextMiddleware)

		app.Middleware.Skip(middleware.IsAuthMiddleware, readyz)
		app.Middleware.Skip(middleware.SetContextMiddleware, readyz)
		app.ANY("/readyz", readyz)

		apiPath := envy.Get("API_PATH", "/api")

		authPath := app.Group(apiPath + "/auth")
		authPath.Middleware.Skip(middleware.IsAuthMiddleware, AuthLoginHandler, AuthGetCerts)
		authPath.Middleware.Skip(middleware.SetContextMiddleware, AuthLoginHandler, AuthGetCerts)
		authPath.POST("/login", AuthLoginHandler)
		authPath.POST("/login/refresh", AuthLoginRefreshHandler)
		authPath.POST("/logout", AuthLogoutHandler)
		authPath.GET("/userinfo", AuthGetUserInfo)
		authPath.GET("/tokeninfo", AuthGetTokenInfo)
		authPath.GET("/validate", AuthGetUserValidate)
		authPath.GET("/certs", AuthGetCerts)

		userPath := app.Group(apiPath + "/user")
		userPath.POST("/", CreateUser)
		userPath.GET("/", GetUsers)
		userPath.PUT("/id/{userId}", UpdateUser)
		userPath.DELETE("/id/{userId}", DeleteUser)

		rolePath := app.Group(apiPath + "/role")
		rolePath.POST("/", CreateRole)
		rolePath.GET("/", GetRoleList)
		rolePath.GET("/name/{roleName}", SearchRolesByName)
		rolePath.GET("/id/{roleId}", GetRoleById)
		rolePath.GET("/policyid/{policyId}", GetRoleByPolicyId)
		rolePath.PUT("/id/{roleId}", UpdateRoleById)
		rolePath.DELETE("/id/{roleId}", DeleteRoleById)

		workspacePath := app.Group(apiPath + "/ws")
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/workspace/{workspaceName}", SearchWorkspacesByName)
		workspacePath.GET("/workspace/id/{workspaceId}", GetWorkspaceById)
		workspacePath.PUT("/workspace/id/{workspaceId}", UpdateWorkspaceById)
		workspacePath.DELETE("/workspace/id/{workspaceId}", DeleteWorkspaceById)

		projectPath := app.Group(apiPath + "/prj")
		projectPath.POST("/", CreateProject)
		projectPath.GET("/", GetProjectList)
		projectPath.GET("/project/{projectName}", SearchProjectsByName)
		projectPath.GET("/project/id/{projectId}", GetProjectById)
		projectPath.PUT("/project/id/{projectId}", UpdateProjectById)
		projectPath.DELETE("/project/id/{projectId}", DeleteProjectById)

		wpmappingPath := app.Group(apiPath + "/wsprj")
		wpmappingPath.POST("/", CreateWPmappings)
		wpmappingPath.GET("/", GetWPmappingListOrderbyWorkspace)
		wpmappingPath.GET("/workspace/id/{workspaceId}", GetWPmappingListByWorkspaceId)
		wpmappingPath.PUT("/", UpdateWPmappings)
		wpmappingPath.DELETE("/workspace/id/{workspaceId}/project/id/{projectId}", DeleteWPmapping)

		workspaceUserRoleMappingPath := app.Group(apiPath + "/wsuserrole")
		workspaceUserRoleMappingPath.POST("/", CreateWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.GET("/", GetWorkspaceUserRoleMappingListOrderbyWorkspace)
		workspaceUserRoleMappingPath.GET("/workspace/id/{workspaceId}", GetWorkspaceUserRoleMappingListByWorkspaceId)
		workspaceUserRoleMappingPath.GET("/user/id/{userId}", GetWorkspaceUserRoleMappingListByUserId)
		workspaceUserRoleMappingPath.GET("/workspace/id/{workspaceId}/user/id/{userId}", GetWorkspaceUserRoleMappingById)
		workspaceUserRoleMappingPath.DELETE("/workspace/id/{workspaceId}/user/id/{userId}", DeleteWorkspaceUserRoleMapping)

		resourcePath := app.Group(apiPath + "/resource")
		resourcePath.POST("/", CreateResources)
		resourcePath.POST("/file/framework/{framework}/menu", CreateMenuResourcesByYaml)
		resourcePath.POST("/file/framework/{framework}", CreateResourcesBySwagger)
		resourcePath.GET("/", GetResources)
		resourcePath.GET("/menus", GetMenuResources)
		resourcePath.PUT("/id/{resourceid}", UpdateResource)
		resourcePath.DELETE("/id/{resourceid}", DeleteResource)
		resourcePath.DELETE("/reset", ResetResource)

		permissionPath := app.Group(apiPath + "/permission")
		permissionPath.POST("/", CreatePermission)
		permissionPath.GET("/", GetPermissions)
		permissionPath.GET("/id/{permissionid}", GetPermission)
		permissionPath.PUT("/id/{permissionid}", UpdatePermission)
		permissionPath.DELETE("/id/{permissionid}", DeletePermission)

		ticketPath := app.Group(apiPath + "/ticket")
		ticketPath.GET("/", GetAllPermissions)
		ticketPath.GET("/menus", GetAllAvailableMenus)
		ticketPath.GET("/framework/{framework}/operationid/{operationid}", GetPermissionTicketByOperationid)

		toolPath := app.Group(apiPath + "/tool")
		toolPath.GET("/mcinfra/sync", SyncProjectListWithMcInfra)

		stsPath := app.Group(apiPath + "/poc" + "/sts")
		stsPath.Use(middleware.IsAuthMiddleware)
		stsPath.GET("/securitykey", AuthSecuritykeyProviderHandler)
	})

	return app
}

func readyz(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"status": "ok"}))
}
