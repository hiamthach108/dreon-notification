package worker

import (
	"context"
	"time"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"go.uber.org/fx"
)

// RunScheduledNotificationWorker polls for due scheduled notifications and publishes them to the send topic until fx stops.
func RunScheduledNotificationWorker(
	lc fx.Lifecycle,
	cfg *config.AppConfig,
	svc service.INotificationSvc,
	log logger.ILogger,
) {
	intervalSec := cfg.Notification.ScheduledPollIntervalSec
	if intervalSec <= 0 {
		intervalSec = 60
	}
	batch := cfg.Notification.ScheduledBatchSize
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
						if err := svc.EnqueueDueScheduledNotifications(workerCtx, batch); err != nil {
							log.Error("scheduled notification enqueue failed", "error", err)
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
