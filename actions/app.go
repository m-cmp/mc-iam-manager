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
		CreateDefaultAdminUserOnIdp()
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

		apiPath := "/api/"

		app.GET("/alive", alive)

		auth := app.Group(apiPath + "auth")
		auth.POST("/login", AuthLoginHandler)
		auth.POST("/login/refresh", AuthLoginRefreshHandler)
		auth.POST("/logout", AuthLogoutHandler)

		auth.GET("/validate", AuthGetUserValidate)
		auth.GET("/userinfo", AuthGetUserInfo)

		auth.GET("/securitykey", AuthGetSecurityKeyHandler)

		auth.GET("/validate", AuthGetUserInfo)

		auth.POST("/user", RegistUser)
		auth.DELETE("/user/{userId}", UnRegistUser)
		auth.GET("/user", GetUserList)
		auth.GET("/user/{userId}", GetUser)
		auth.PATCH("/user/{userId}", UpdateUserProfile)

		auth.POST("/usergroup", CreateUserGroup)
		auth.PATCH("/usergroup/{groupId}", UpdateUserGroup)
		auth.GET("/usergroup", GetUserGroupList)
		auth.GET("/usergroup/{groupId}", GetUserGroup)
		auth.DELETE("/usergroup/{groupId}", DeleteUserGroup)

		// manage := app.Group(apiPath + "manage")
		// manage.POST("/login", GetWorkspace)
		// manage.GET("/logout", GetWorkspace)

		// auth := app.Group(apiPath)
		// auth.Middleware.Skip(IsAuth, IamLoginApi)
		// auth.POST("/login", IamLoginApi)

		// userPath := app.Group(apiPath + "users")
		// userPath.GET("/", GetUsersList)

		rolePath := app.Group(apiPath + "/auth/role")
		rolePath.GET("/", GetUserRoleList)
		rolePath.GET("/{roleId}", GetUserRole)
		rolePath.PATCH("/{roleId}", UpdateUserRole)
		rolePath.POST("/", CreateUserRole)
		rolePath.DELETE("/{roleId}", DeleteUserRole)

		workspacePath := app.Group(apiPath + "/ws/workspace")
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/{workspaceId}", GetWorkspace)
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.DELETE("/{workspaceId}", DeleteWorkspace)
		workspacePath.PATCH("/{workspaceId}", UpdateWorkspace)
		workspacePath.GET("/{workspaceId}/project", AttachedProjectByWorkspace)

		workspacePath.POST("/{workspaceId}/attachproject", AttachProjectToWorkspace)
		workspacePath.DELETE("/{workspaceId}/attachproject/{projectId}", DeleteProjectFromWorkspace)
		workspacePath.POST("/{workspaceId}/assigneduser", AssignUserToWorkspace)

		workspaceUserPath := app.Group(apiPath + "/ws/user")
		workspaceUserPath.GET("/{userId}", GetWorkspaceListByUser)

		// mappingPath := app.Group(apiPath + "mapping")
		// mappingPath.POST("/ws/user", MappingWsUser)
		// mappingPath.POST("/ws/user/role", MappingWsUserRole)
		// mappingPath.POST("/ws/project", AttachProjectToWorkspace)
		// mappingPath.GET("/ws/id/{workspaceId}/project", MappingGetProjectByWorkspace)
		// mappingPath.GET("/ws/id/{workspaceId}/project/id/{projectId}", MappingWsProjectValidCheck)
		// mappingPath.DELETE("/ws/project", MappingDeleteWsProject)
		// mappingPath.GET("/user/id/{userId}/workspace", MappingGetWsUserRole)

		projectPath := app.Group(apiPath + "/ws/project")
		projectPath.GET("/{projectId}", GetProject)
		projectPath.GET("/", GetProjectList)
		projectPath.POST("/", CreateProject)
		projectPath.DELETE("/{projectId}", DeleteProject)
		projectPath.PATCH("/{projectId}", UpdateProject)

		// debugPath := app.Group("/debug")
		// debugPath.GET("/getrealmrolebyid/{roleid}", DebugGetRealmRoleByID)
		// debugPath.GET("/kc/kchomehandler", KcHomeHandler)
		// debugPath.GET("/kc/kccreateuserhandler", KcCreateUserHandler)
		debugPath := app.Group("/debug")
		debugPath.GET("/init", InitApi)
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
