package events

import (
	"context"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/internal/shared/constant"
	"go.uber.org/fx"
)

// RunRouter registers topic subscriptions and runs the router. Add further topics here as needed.
func RunRouter(lc fx.Lifecycle, router *message.Router, subscriber *amqp.Subscriber, service service.INotificationSvc) {
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
