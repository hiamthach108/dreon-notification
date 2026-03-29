package main

import (
	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/cache"
	"github.com/hiamthach108/dreon-notification/pkg/database"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/fcm"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
	"github.com/hiamthach108/dreon-notification/presentation/events"
	grpcserver "github.com/hiamthach108/dreon-notification/presentation/grpc"
	"github.com/hiamthach108/dreon-notification/presentation/http"
	"github.com/hiamthach108/dreon-notification/presentation/http/handler"
	"github.com/hiamthach108/dreon-notification/presentation/worker"
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
			email.NewRenderer,
			sms.NewMockClient, // TODO: remove this and use NewTwilioClient when Twilio is configured
			sms.NewBodyRenderer,
			fcm.NewClient,

			// Events
			events.NewLoggerAdapter,
			events.NewAMQPPublisher,
			events.NewAMQPSubscriber,

			// Repositories
			repository.NewNotificationRepository,
			repository.NewPushTopicRepository,
			repository.NewMailboxRepository,
			repository.NewUserFCMTokenRepository,

			// Services
			service.NewNotificationSvc,
			service.NewPushTopicSvc,
			service.NewMailboxSvc,
			service.NewUserFCMTokenSvc,

			// Handlers
			handler.NewNotificationHandler,
			handler.NewPushTopicHandler,
			handler.NewUserFCMTokenHandler,

			// gRPC server (NotiInternal: notification management)
			grpcserver.NewNotiInternalServer,
			grpcserver.NewGRPCServer,
		),
		fx.Invoke(http.RegisterHooks),
		fx.Invoke(grpcserver.RegisterHooks),
		fx.Invoke(events.RunConsumers),
		fx.Invoke(worker.RunPendingRetryWorker),
		fx.Invoke(worker.RunScheduledNotificationWorker),
	)

	app.Run()
}
