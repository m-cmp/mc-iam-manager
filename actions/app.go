package actions

import (
	"net/http"

	"mc_iam_manager/locales"
	"mc_iam_manager/models"
	"mc_iam_manager/public"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v3/pop/popmw"
	"github.com/gobuffalo/envy"
	forcessl "github.com/gobuffalo/mw-forcessl"
	i18n "github.com/gobuffalo/mw-i18n/v2"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/unrolled/secure"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")

var (
	app *buffalo.App
	T   *i18n.Translator
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
	if app == nil {
		app = buffalo.New(buffalo.Options{
			Env:         ENV,
			SessionName: "_gocloak_session",
		})

		// Automatically redirect to SSL
		app.Use(forceSSL())

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Protect against CSRF attacks. https://www.owasp.org/index.php/Cross-Site_Request_Forgery_(CSRF)
		// Remove to disable this.
		// app.Use(csrf.New)

		// Wraps each request in a transaction.
		//   c.Value("tx").(*pop.Connection)
		// Remove to disable this.
		app.Use(popmw.Transaction(models.DB))
		// Setup and use translations:
		app.Use(translations())

		// kc := app.Group("/mcloak")
		// kc.GET("/home", KcHomeHandler) // /mcloak/home
		// kc.GET("/login", KcLoginHandler)
		// kc.GET("/createuser", KcCreateUserHandler)

		// bf := app.Group("/iam")
		// bf.Use(IsAuth)
		// bf.Middleware.Skip(IsAuth, IamLoginForm, IamLogin, NotAuthUserTestPageHandler)
		// bf.GET("/login", IamLoginForm)
		// bf.POST("/login", IamLogin)
		// bf.GET("/authuser/not", NotAuthUserTestPageHandler)

		// bf.GET("/", HomeHandler)
		// bf.GET("/authuser", AuthUserTestPageHandler)

		// app.GET("/saml/aws", AwsSamlSTSKey)
		// app.GET("/saml/ali", AliSamlSTSKey)

		//app.Use(IsAuth)

		apiPath := "/api/v1/"

		auth := app.Group(apiPath)
		auth.Middleware.Skip(IsAuth, IamLoginApi)
		auth.POST("/login", IamLoginApi)

		rolePath := app.Group(apiPath + "roles")
		rolePath.GET("/", ListRole)
		rolePath.GET("/id/{roleId}", GetRole)
		//rolePath.GET("/user/id/{userId}", GetRoleByUser)
		rolePath.PUT("/id/{roleId}", UpdateRole)
		rolePath.POST("/", CreateRole)
		rolePath.DELETE("/id/{roleId}", DeleteRole)

		workspacePath := app.Group(apiPath + "workspace")
		workspacePath.GET("/", GetWorkspaceList)
		workspacePath.GET("/id/{workspaceId}", GetWorkspace)
		workspacePath.POST("/", CreateWorkspace)
		workspacePath.DELETE("/id/{workspaceId}", DeleteWorkspace)

		mappingPath := app.Group(apiPath + "mapping")
		mappingPath.POST("/ws/user", MappingWsUser)
		mappingPath.POST("/ws/user/role", MappingWsUserRole)
		mappingPath.POST("/ws/project", MappingWsProject)
		mappingPath.GET("/ws/id/{workspaceId}/project", MappingGetProjectByWorkspace)
		mappingPath.GET("/ws/id/{workspaceId}/project/id/{projectId}", MappingWsProjectValidCheck)
		mappingPath.DELETE("/ws/project", MappingDeleteWsProject)

		projectPath := app.Group(apiPath + "project")
		projectPath.GET("/id/{projectId}", GetProject)
		projectPath.GET("/", GetProjectList)
		projectPath.POST("/", CreateProject)
		projectPath.DELETE("/id/{projectId}", DeleteProject)

		app.ServeFiles("/", http.FS(public.FS())) // serve files from the public directory
	}

	return app
}

// translations will load locale files, set up the translator `actions.T`,
// and will return a middleware to use to load the correct locale for each
// request.
// for more information: https://gobuffalo.io/en/docs/localization
func translations() buffalo.MiddlewareFunc {
	var err error
	if T, err = i18n.New(locales.FS(), "en-US"); err != nil {
		app.Stop(err)
	}
	return T.Middleware()
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
