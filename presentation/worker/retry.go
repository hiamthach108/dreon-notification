package worker

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"go.uber.org/fx"
)

// RunPendingRetryWorker starts a ticker that claims due PENDING rows and publishes them to the retry topic until fx stops.
func RunPendingRetryWorker(
	lc fx.Lifecycle,
	cfg *config.AppConfig,
	svc service.INotificationSvc,
	log logger.ILogger,
	publisher *amqp.Publisher,
) {
	intervalSec := cfg.Notification.RetryIntervalSec
	if intervalSec <= 0 {
		intervalSec = 60
	}
	batch := cfg.Notification.RetryBatchSize
	if batch <= 0 {
		batch = 10
	}
	interval := time.Duration(intervalSec) * time.Second

	var cancel context.CancelFunc

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			workerCtx, done := context.WithCancel(context.Background())
			cancel = done
			go func() {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-workerCtx.Done():
						return
					case <-ticker.C:
						if err := svc.EnqueuePendingRetries(workerCtx, batch); err != nil {
							log.Error("pending notification retry enqueue failed", "error", err)
						}
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}
			return nil
		},
	})
}
