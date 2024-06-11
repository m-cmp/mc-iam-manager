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
		rolePath.GET("/{roleId}", GetRole)
		rolePath.DELETE("/{roleId}", DeleteRole)

		workspacePath := app.Group(apiPath + "ws")
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/workspace/{workspaceId}", GetWorkspace)
		workspacePath.PUT("/workspace/{workspaceId}", UpdateWorkspace)
		workspacePath.DELETE("/workspace/{workspaceId}", DeleteWorkspace)

		projectPath := app.Group(apiPath + "prj")
		projectPath.POST("/", CreateProject)
		projectPath.GET("/", GetProjectList)
		projectPath.GET("/project/{projectId}", GetProject)
		projectPath.PUT("/project/{projectId}", UpdateProject)
		projectPath.DELETE("/project/{projectId}", DeleteProject)

		workspaceProjectMappingPath := app.Group(apiPath + "wsprj")
		workspaceProjectMappingPath.POST("/workspace/{workspaceId}", CreateWorkspaceProjectMapping)
		workspaceProjectMappingPath.GET("/", GetWorkspaceProjectMappingList)
		workspaceProjectMappingPath.GET("/workspace/{workspaceId}", GetWorkspaceProjectMappingByWorkspace)
		workspaceProjectMappingPath.PUT("/workspace/{workspaceId}", UpdateWorkspaceProjectMapping)
		workspaceProjectMappingPath.DELETE("/workspace/{workspaceId}/project/{projectId}", DeleteWorkspaceProjectMapping)
		workspaceProjectMappingPath.DELETE("/workspace/{workspaceId}", DeleteWorkspaceProjectMappingAllByWorkspace)
		workspaceProjectMappingPath.DELETE("/project/{projectId}", DeleteWorkspaceProjectMappingByProject)

		workspaceUserRoleMappingPath := app.Group(apiPath + "wsuserrole")
		workspaceUserRoleMappingPath.POST("/workspace/{workspaceId}", CreateWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.GET("/", GetWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.GET("/workspace/{workspaceId}", GetWorkspaceUserRoleMappingByWorkspace)
		workspaceUserRoleMappingPath.GET("/workspace/{workspaceId}/user/{userId}", GetWorkspaceUserRoleMappingByWorkspaceUser)
		workspaceUserRoleMappingPath.GET("/user/{userId}", GetWorkspaceUserRoleMappingByUser)
		workspaceUserRoleMappingPath.PUT("/workspace/{workspaceId}/user/{userId}", UpdateWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.DELETE("/workspace/{workspaceId}/user/{userId}", DeleteWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.DELETE("/workspace/{workspaceId}", DeleteWorkspaceUserRoleMappingAll)
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
