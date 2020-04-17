package server

import (
	"bastion/internal/auth"
	"bastion/internal/datastore"
	"bastion/internal/log"
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"html/template"
	"io"
	"net/http"
	"os"
)

type BastionServer struct {
	config     ConfigStruct
	oidcClient *auth.OIDCClient
	keySet     oidc.KeySet
	web        *echo.Echo
	templates  *template.Template
	sessions   sessions.Store
	logger     *zap.Logger
}

const (
	requestLoggerContextKey = "requestLogger"
)

func New() BastionServer {
	config := LoadConfiguration()

	appLogger, err := log.Init(config.Log.Level)
	if err != nil {
		fmt.Printf("Error initializing logging: %s\n", err)
		os.Exit(1)
	}

	datastore.Configure(datastore.Config{
		Logger:         appLogger,
		DataSourceName: config.Datastore.DataSourceName})
	if err := datastore.Open(); err != nil {
		appLogger.Fatal(err.Error())
	}

	t, err := template.ParseGlob(config.Web.TemplatesDir + "/*.html")
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	app := BastionServer{
		config:    config,
		web:       echo.New(),
		templates: t,
		logger:    appLogger,
	}
	app.oidcClient, err = auth.New(
		app.config.OIDC.Issuer,
		app.config.OIDC.ClientID,
		app.config.OIDC.ClientSecret,
		app.config.OIDC.RedirectURL,
		[]string{})
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	// TODO jwks_uri нужно автоматически получать из discovery (.well-known/openid-configuration)
	app.keySet = oidc.NewRemoteKeySet(context.Background(), "https://idp.example.com/discovery/keys")

	app.web.HideBanner = true
	app.web.Debug = true
	app.web.Renderer = &app
	app.web.Pre(middleware.HTTPSRedirect())
	if app.config.Log.PrintRequests {
		app.web.Pre(LoggerMiddleware(appLogger))
	}
	app.web.Pre(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll:   true,
		DisablePrintStack: true,
	}))
	app.web.Pre(middleware.RemoveTrailingSlash())
	app.web.Pre(app.XRequestIDMiddleware)

	app.web.Static("/", config.Web.StaticContentDir)
	app.web.GET("/", func(context echo.Context) error {
		return context.Redirect(http.StatusMovedPermanently, "/app/main")
	})

	front := app.web.Group("/app")
	front.Use(app.OIDCMiddleware)
	front.GET("/main", app.indexHandler)

	auth := app.web.Group("/auth")
	auth.GET("/callback", app.authCallback)
	auth.GET("/logout", app.logoutHandler)

	api := app.web.Group("/api")
	api.Use(app.OIDCMiddleware)
	api.GET("/userdata", app.readUserData)

	api.POST("/sessions", app.createSessionHandler)
	api.GET("/sessions/:token", app.readSessionHandler)
	//api.DELETE("/sessions/:token", app.DeleteSessionHandler)

	api.POST("/sessiontemplates", app.createSessionTemplateHandler)
	api.DELETE("/sessiontemplates/:id", app.deleteSessionTemplateHandler)

	app.initSessionStore()
	return app
}

func (app *BastionServer) Run() {
	app.logger.Info("Bastion server listening", zap.String("address", app.config.BindAddress))
	app.web.Logger.Fatal(app.web.StartTLS(app.config.BindAddress, app.config.TLS.CertificateFile, app.config.TLS.KeyFile))
}

func (app *BastionServer) Shutdown() {
	if err := datastore.Close(); err != nil {
		app.logger.Error(err.Error())
	}
	app.logger.Info("Shutdown")
	_ = app.logger.Sync()
}

func (app *BastionServer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return app.templates.ExecuteTemplate(w, name, data)
}
