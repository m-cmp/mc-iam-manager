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
		auth := app.Group(apiPath + "auth")
		auth.ANY("/{path:.+}", buffalo.WrapHandlerFunc(mcimw.BeginAuthHandler))

		alive := app.Group("/alive")
		alive.GET("/", aliveSig)
		mcimw.GrantedRoleList = []string{"admin"}
		alive.GET("/admin", mcimw.BuffaloMcimw(aliveSig))

		mcimw.GrantedRoleList = []string{"viewer"}
		alive.GET("/viewer", mcimw.BuffaloMcimw(aliveSig))

		mcimw.GrantedRoleList = []string{"operator"}
		alive.GET("/operator", mcimw.BuffaloMcimw(aliveSig))

		sts := app.Group(apiPath + "poc" + "/sts")
		mcimw.GrantedRoleList = []string{}
		sts.GET("/securitykey", mcimw.BuffaloMcimw(AuthSecuritykeyProviderHandler))

		mcimw.AuthMethod = mcimw.EnvKeycloak
		mcimw.GrantedRoleList = []string{}
		app.Use(mcimw.BuffaloMcimw)
		rolePath := app.Group(apiPath + "role")
		rolePath.POST("/", CreateRole)
		rolePath.GET("/", GetRoleList)
		rolePath.GET("/role/{roleName}", SearchRolesByName)
		rolePath.GET("/role/id/{roleId}", GetRoleById)
		rolePath.PUT("/role/id/{roleId}", UpdateRoleById)
		rolePath.DELETE("/role/id/{roleId}", DeleteRoleById)

		workspacePath := app.Group(apiPath + "ws")
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/workspace/{workspaceName}", SearchWorkspacesByName)
		workspacePath.GET("/workspace/id/{workspaceId}", GetWorkspaceById)
		workspacePath.PUT("/workspace/id/{workspaceId}", UpdateWorkspaceById)
		workspacePath.DELETE("/workspace/id/{workspaceId}", DeleteWorkspaceById)

		projectPath := app.Group(apiPath + "prj")
		projectPath.POST("/", CreateProject)
		projectPath.GET("/", GetProjectList)
		projectPath.GET("/project/{projectName}", SearchProjectsByName)
		projectPath.GET("/project/id/{projectId}", GetProjectById)
		projectPath.PUT("/project/id/{projectId}", UpdateProjectById)
		projectPath.DELETE("/project/id/{projectId}", DeleteProjectById)

		wpmappingPath := app.Group(apiPath + "wsprj")
		wpmappingPath.POST("/", CreateWPmappings)
		wpmappingPath.GET("/", GetWPmappingListOrderbyWorkspace)
		wpmappingPath.GET("/workspace/id/{workspaceId}", GetWPmappingListByWorkspaceId)
		wpmappingPath.PUT("/", UpdateWPmappings)
		wpmappingPath.DELETE("/workspace/id/{workspaceId}/project/id/{projectId}", DeleteWPmapping)

		workspaceUserRoleMappingPath := app.Group(apiPath + "wsuserrole")
		workspaceUserRoleMappingPath.POST("/", CreateWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.GET("/", GetWorkspaceUserRoleMappingListOrderbyWorkspace)
		workspaceUserRoleMappingPath.GET("/workspace/id/{workspaceId}", GetWorkspaceUserRoleMappingListByWorkspaceId)
		workspaceUserRoleMappingPath.GET("/user/id/{userId}", GetWorkspaceUserRoleMappingListByUserId)
		workspaceUserRoleMappingPath.GET("/workspace/id/{workspaceId}/user/id/{userId}", GetWorkspaceUserRoleMappingById)
		workspaceUserRoleMappingPath.DELETE("/workspace/id/{workspaceId}/user/id/{userId}", DeleteWorkspaceUserRoleMapping)

		tool := app.Group(apiPath + "tool")
		tool.GET("/mcinfra/sync", SyncProjectListWithMcInfra)
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
