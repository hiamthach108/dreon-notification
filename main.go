package main

import (
	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/cache"
	"github.com/hiamthach108/dreon-notification/pkg/database"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	grpcserver "github.com/hiamthach108/dreon-notification/presentation/grpc"
	"github.com/hiamthach108/dreon-notification/presentation/http"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func main() {
	app := fx.New(
		fx.WithLogger(func(appLogger logger.ILogger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: appLogger.GetZapLogger()}
		}),
		fx.Provide(
			// Core
			config.NewAppConfig,
			logger.NewLogger,
			cache.NewAppCache,
			database.NewDbClient,
			http.NewHttpServer,
			email.NewResendEmailClient,
			func(cfg *config.AppConfig) email.IRenderer {
				dir := cfg.Email.TemplateDir
				if dir == "" {
					dir = "templates/emails"
				}
				return email.NewRenderer(email.WithTemplateDir(dir))
			},

			// Repositories
			repository.NewNotificationRepository,

			// Services
			service.NewNotificationSvc,

			// gRPC server (NotiInternal: notification management)
			grpcserver.NewNotiInternalServer,
			grpcserver.NewGRPCServer,
		),
		fx.Invoke(http.RegisterHooks),
		fx.Invoke(grpcserver.RegisterHooks),
	)

	app.Run()
}
