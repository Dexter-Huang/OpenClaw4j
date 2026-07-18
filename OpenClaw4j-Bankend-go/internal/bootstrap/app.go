package bootstrap

import (
	"errors"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao/pgdao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
	authmiddleware "github.com/seaskyland/openclaw4j-backend-go/internal/transport/hertz/middleware"
	hertzserver "github.com/seaskyland/openclaw4j-backend-go/internal/transport/hertz/server"
)

var ErrMissingDatabase = errors.New("database is required when authenticator is not provided")

type Options struct {
	DB                    db.DBTX
	TokenSessions         dao.TokenSessionDAO
	HertzOptions          []config.Option
	Authenticator         authmiddleware.Authenticator
	AuthService           *auth.Service
	LegacyAPIKeyEncryptor auth.LegacyAPIKeyEncryptor
	RegisterConsoleRoutes func(group *route.RouterGroup)
	RegisterAPIRoutes     func(group *route.RouterGroup)
}

type App struct {
	HTTP          *server.Hertz
	Authenticator authmiddleware.Authenticator
	AuthService   *auth.Service
}

func New(options Options) (*App, error) {
	authService := options.AuthService
	if authService == nil && options.DB != nil {
		authService = buildAuthService(options.DB, options.TokenSessions, options.LegacyAPIKeyEncryptor)
	}

	authenticator := options.Authenticator
	if authenticator == nil {
		if authService == nil {
			if options.DB == nil {
				return nil, ErrMissingDatabase
			}
			authService = buildAuthService(options.DB, options.TokenSessions, options.LegacyAPIKeyEncryptor)
		}
		authenticator = authService
	}
	if authService == nil {
		if service, ok := authenticator.(*auth.Service); ok {
			authService = service
		}
	}
	if authenticator == nil {
		if options.DB == nil {
			return nil, ErrMissingDatabase
		}
		authenticator = buildAuthService(options.DB, options.TokenSessions, options.LegacyAPIKeyEncryptor)
		if authService == nil {
			if service, ok := authenticator.(*auth.Service); ok {
				authService = service
			}
		}
	}

	httpServer := hertzserver.New(hertzserver.Options{
		HertzOptions:          options.HertzOptions,
		Authenticator:         authenticator,
		SessionIssuer:         authService,
		RegisterConsoleRoutes: options.RegisterConsoleRoutes,
		RegisterAPIRoutes:     options.RegisterAPIRoutes,
	})

	return &App{
		HTTP:          httpServer,
		Authenticator: authenticator,
		AuthService:   authService,
	}, nil
}

func buildAuthService(database db.DBTX, tokenSessions dao.TokenSessionDAO, legacyAPIKeyEncryptor auth.LegacyAPIKeyEncryptor) *auth.Service {
	queries := db.New(database)
	return auth.NewService(auth.ServiceOptions{
		Accounts:              pgdao.NewAccountDAO(queries),
		Workspaces:            pgdao.NewWorkspaceDAO(queries),
		APIKeys:               pgdao.NewAPIKeyDAO(queries),
		TokenSessions:         tokenSessions,
		LegacyAPIKeyEncryptor: legacyAPIKeyEncryptor,
	})
}
