package events

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/internal/shared/constant"
	"go.uber.org/fx"
)

// RunConsumers registers topic subscriptions and runs the router. Add further topics here as needed.
func RunConsumers(lc fx.Lifecycle, subscriber *amqp.Subscriber, service service.INotificationSvc, log watermill.LoggerAdapter) {

	router, err := message.NewRouter(message.RouterConfig{}, log)
	if err != nil {
		log.Error("failed to create router", err, watermill.LogFields{"error": err})
		return
	}
	// Register consumer handler for the notification send topic
	router.AddConsumerHandler("notification_send_handler", constant.EventTopicNotificationsSend, subscriber, service.ProcessNotificationFromQueue)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				_ = router.Run(context.Background())
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return router.Close()
		},
	})
}
