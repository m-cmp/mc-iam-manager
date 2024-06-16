package actions

import (
	"mc_iam_manager/mcimw"
	"mc_iam_manager/models"
	"net/http"
	"sync"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v3/pop/popmw"
	"github.com/gobuffalo/envy"
	contenttype "github.com/gobuffalo/mw-contenttype"
	forcessl "github.com/gobuffalo/mw-forcessl"
	i18n "github.com/gobuffalo/mw-i18n/v2"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"
	"github.com/unrolled/secure"
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
			SessionName: "_mc_iam_manager_session",
		})

		app.Use(forceSSL())
		app.Use(paramlogger.ParameterLogger)
		app.Use(contenttype.Set("application/json"))
		app.Use(popmw.Transaction(models.DB))

		apiPath := envy.Get("API_PATH", "/api/")

		mcimw.AuthMethod = mcimw.EnvKeycloak

		alive := app.Group("/alive")
		alive.GET("/", aliveSig)
		mcimw.GrantedRoleList = []string{"admin"}
		alive.GET("/admin", mcimw.BuffaloMcimw(aliveSig))

		mcimw.GrantedRoleList = []string{"viewer"}
		alive.GET("/viewer", mcimw.BuffaloMcimw(aliveSig))

		mcimw.GrantedRoleList = []string{"operator"}
		alive.GET("/operator", mcimw.BuffaloMcimw(aliveSig))

		auth := app.Group(apiPath + "auth")
		auth.ANY("/{path:.+}", buffalo.WrapHandlerFunc(mcimw.BeginAuthHandler))

		sts := app.Group(apiPath + "sts")
		sts.GET("/securitykey", AuthGetSecurityKeyHandler)

		rolePath := app.Group(apiPath + "role")
		rolePath.POST("/", CreateRole)
		rolePath.GET("/", GetRoleList)
		rolePath.GET("/role/{roleName}", GetRoleByName)
		rolePath.GET("/role/id/{roleId}", GetRoleById)
		rolePath.PUT("/role/{roleName}", UpdateRoleByName)
		rolePath.PUT("/role/id/{roleId}", UpdateRoleById)
		rolePath.DELETE("/role/{roleName}", DeleteRoleByName)
		rolePath.DELETE("/role/id/{roleId}", DeleteRoleById)

		workspacePath := app.Group(apiPath + "ws")
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/workspace/{workspaceName}", GetWorkspaceByName)
		workspacePath.GET("/workspace/id/{workspaceId}", GetWorkspaceById)
		workspacePath.PUT("/workspace/{workspaceName}", UpdateWorkspaceByName)
		workspacePath.PUT("/workspace/id/{workspaceId}", UpdateWorkspaceById)
		workspacePath.DELETE("/workspace/{workspaceName}", DeleteWorkspaceByName)
		workspacePath.DELETE("/workspace/id/{workspaceId}", DeleteWorkspaceById)

		projectPath := app.Group(apiPath + "prj")
		projectPath.POST("/", CreateProject)
		projectPath.GET("/", GetProjectList)
		projectPath.GET("/project/{projectName}", GetProjectByName)
		projectPath.GET("/project/id/{projectId}", GetProjectById)
		projectPath.PUT("/project/{projectName}", UpdateProjectByName)
		projectPath.PUT("/project/id/{projectId}", UpdateProjectById)
		projectPath.DELETE("/project/{projectName}", DeleteProjectByName)
		projectPath.DELETE("/project/id/{projectId}", DeleteProjectById)

		workspaceProjectMappingPath := app.Group(apiPath + "wsprj")
		workspaceProjectMappingPath.POST("/workspace/{workspaceName}", CreateWorkspaceProjectMappingByName)
		workspaceProjectMappingPath.POST("/workspace/id/{workspaceId}", CreateWorkspaceProjectMappingById)
		workspaceProjectMappingPath.GET("/", GetWorkspaceProjectMappingListByWorkspace)
		workspaceProjectMappingPath.GET("/workspace/{workspaceName}", GetWorkspaceProjectMappingByWorkspaceName)
		workspaceProjectMappingPath.GET("/workspace/id/{workspaceId}", GetWorkspaceProjectMappingByWorkspaceId)
		workspaceProjectMappingPath.DELETE("/workspace/{workspaceName}/project/{projectName}", DeleteWorkspaceProjectMappingByName)
		workspaceProjectMappingPath.DELETE("/workspace/id/{workspaceId}/project/id/{projectId}", DeleteWorkspaceProjectMappingById)

		workspaceUserRoleMappingPath := app.Group(apiPath + "wsuserrole")
		workspaceUserRoleMappingPath.POST("/workspace/{workspaceName}", CreateWorkspaceUserRoleMappingByName)
		workspaceUserRoleMappingPath.GET("/", GetWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.GET("/workspace/{workspaceName}", GetWorkspaceUserRoleMappingByWorkspaceName)
		workspaceUserRoleMappingPath.GET("/workspace/id/{workspaceId}", GetWorkspaceUserRoleMappingByWorkspacId)
		workspaceUserRoleMappingPath.GET("/user/{userId}", GetWorkspaceUserRoleMappingByUser)
		workspaceUserRoleMappingPath.DELETE("/workspace/{workspaceName}/user/{userId}", DeleteWorkspaceUserRoleMappingByName)
		workspaceUserRoleMappingPath.DELETE("/workspace/id/{workspaceId}/user/{userId}", DeleteWorkspaceUserRoleMappingById)
		// workspaceUserRoleMappingPath.DELETE("/workspace/id/{workspaceId}/user/{userId}", DeleteWorkspaceUserRoleMappingByName)
	})

	return app
}

func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}

func aliveSig(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"ststus": "ok"}))
}
