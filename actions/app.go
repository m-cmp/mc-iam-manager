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
		rolePath.GET("/role/{roleName}", SearchRolesByName)
		rolePath.GET("/role/id/{roleUUID}", GetRoleByUUID)
		rolePath.PUT("/role/id/{roleUUID}", UpdateRoleByUUID)
		rolePath.DELETE("/role/id/{roleUUID}", DeleteRoleByUUID)

		workspacePath := app.Group(apiPath + "ws")
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/workspace/{workspaceName}", SearchWorkspacesByName)
		workspacePath.GET("/workspace/id/{workspaceUUID}", GetWorkspaceByUUID)
		workspacePath.PUT("/workspace/id/{workspaceUUID}", UpdateWorkspaceByUUID)
		workspacePath.DELETE("/workspace/id/{workspaceUUID}", DeleteWorkspaceByUUID)

		projectPath := app.Group(apiPath + "prj")
		projectPath.POST("/", CreateProject)
		projectPath.GET("/", GetProjectList)
		projectPath.GET("/project/{projectName}", SearchProjectsByName)
		projectPath.GET("/project/id/{projectUUID}", GetProjectByUUID)
		projectPath.PUT("/project/id/{projectUUID}", UpdateProjectByUUID)
		projectPath.DELETE("/project/id/{projectUUID}", DeleteProjectByUUID)

		wpmappingPath := app.Group(apiPath + "wsprj")
		wpmappingPath.POST("/", CreateWPmappings)
		wpmappingPath.GET("/", GetWPmappingListOrderbyWorkspace)
		wpmappingPath.GET("/workspace/id/{workspaceUUID}", GetWPmappingListByWorkspaceUUID)
		wpmappingPath.PUT("/", UpdateWPmappings)
		wpmappingPath.DELETE("/workspace/id/{workspaceUUID}/project/id/{projectUUID}", DeleteWPmapping)

		workspaceUserRoleMappingPath := app.Group(apiPath + "wsuserrole")
		workspaceUserRoleMappingPath.POST("/", CreateWorkspaceUserRoleMapping)
		workspaceUserRoleMappingPath.GET("/", GetWorkspaceUserRoleMappingListOrderbyWorkspace)
		workspaceUserRoleMappingPath.GET("/workspace/id/{workspaceUUID}", GetWorkspaceUserRoleMappingListByWorkspaceUUID)
		workspaceUserRoleMappingPath.GET("/user/id/{userId}", GetWorkspaceUserRoleMappingListByUserId)
		workspaceUserRoleMappingPath.DELETE("/workspace/id/{workspaceUUID}/user/id/{userId}", DeleteWorkspaceUserRoleMapping)
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
