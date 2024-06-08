package actions

import (
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

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")

var (
	app     *buffalo.App
	appOnce sync.Once
	T       *i18n.Translator
)

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
//
// Routing, middleware, groups, etc... are declared TOP -> DOWN.
// This means if you add a middleware to `app` *after* declaring a
// group, that group will NOT have that new middleware. The same
// is true of resource declarations as well.
//
// It also means that routes are checked in the order they are declared.
// `ServeFiles` is a CATCH-ALL route, so it should always be
// placed last in the route declarations, as it will prevent routes
// declared after it to never be called.
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

		// Automatically redirect to SSL
		app.Use(forceSSL())

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Set the request content type to JSON
		app.Use(contenttype.Set("application/json"))

		// Wraps each request in a transaction.
		//   c.Value("tx").(*pop.Connection)
		// Remove to disable this.
		app.Use(popmw.Transaction(models.DB))

		app.GET("/alive", alive)

		apiPath := "/api/"

		auth := app.Group(apiPath + "auth")
		auth.POST("/login", AuthLoginHandler)
		auth.POST("/login/refresh", AuthLoginRefreshHandler)
		auth.POST("/logout", AuthLogoutHandler)
		auth.GET("/userinfo", AuthGetUserInfo)
		auth.GET("/validate", AuthGetUserValidate)
		auth.GET("/securitykey", AuthGetSecurityKeyHandler)

		rolePath := app.Group(apiPath + "/role")
		rolePath.POST("/", CreateRole)
		rolePath.GET("/", GetRoleList)
		rolePath.GET("/{roleId}", GetRole)
		// rolePath.PUT("/{roleId}", UpdateRole)
		rolePath.DELETE("/{roleId}", DeleteRole)

		workspacePath := app.Group(apiPath + "/ws")
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/workspace/{workspaceId}", GetWorkspace)
		workspacePath.PUT("/workspace/{workspaceId}", UpdateWorkspace)
		workspacePath.DELETE("/workspace/{workspaceId}", DeleteWorkspace)

		projectPath := app.Group(apiPath + "/prj")
		projectPath.POST("/", CreateProject)
		projectPath.GET("/", GetProjectList)
		projectPath.GET("/project/{projectId}", GetProject)
		projectPath.PUT("/project/{projectId}", UpdateProject)
		projectPath.DELETE("/project/{projectId}", DeleteProject)

		workspaceProjectMappingPath := app.Group(apiPath + "/wsprj")
		workspaceProjectMappingPath.POST("/workspace/{workspaceId}", CreateWorkspaceProjectMapping)
		workspaceProjectMappingPath.GET("/", GetWorkspaceProjectMappingList)
		workspaceProjectMappingPath.GET("/workspace/{workspaceId}", GetWorkspaceProjectMappingByWorkspace)
		workspaceProjectMappingPath.PUT("/workspace/{workspaceId}", UpdateWorkspaceProjectMapping)
		workspaceProjectMappingPath.DELETE("/workspace/{workspaceId}/project/{projectId}", DeleteWorkspaceProjectMapping)
		workspaceProjectMappingPath.DELETE("/workspace/{workspaceId}", DeleteWorkspaceProjectMappingAllByWorkspace)
		workspaceProjectMappingPath.DELETE("/project/{projectId}", DeleteWorkspaceProjectMappingByProject)

		// workspaceUserRoleMappingPath := app.Group(apiPath + "/wsuserrole")
		// workspaceUserRoleMappingPath.POST("/workspace/{workspaceId}", CreateWorkspaceUserRoleMapping)
		// workspaceUserRoleMappingPath.GET("/", GetWorkspaceUserRoleMappingList)
		// workspaceUserRoleMappingPath.GET("/workspace/{workspaceId}", GetWorkspaceUserRoleMappingByWorkspace)
		// workspaceUserRoleMappingPath.GET("/user/{userId}", GetWorkspaceUserRoleMappingByUser)
		// workspaceUserRoleMappingPath.PUT("/workspace/{workspaceId}/user/{userId}", UpdateWorkspaceUserRoleMapping)
		// workspaceUserRoleMappingPath.DELETE("/workspace/{workspaceId}/user/{userId}", DeleteWorkspaceUserRoleMapping)
		// workspaceUserRoleMappingPath.DELETE("/workspace/{workspaceId}", DeleteWorkspaceProjectMappingAll)
	})

	return app
}

// forceSSL will return a middleware that will redirect an incoming request
// if it is not HTTPS. "http://example.com" => "https://example.com".
// This middleware does **not** enable SSL. for your application. To do that
// we recommend using a proxy: https://gobuffalo.io/en/docs/proxy
// for more information: https://github.com/unrolled/secure/
func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}

func alive(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"ststus": "ok"}))
}
