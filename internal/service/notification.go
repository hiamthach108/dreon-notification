package service

import (
	"context"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
)

type INotificationSvc interface {
	SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error)
}

type NotificationSvc struct {
	logger logger.ILogger
	repo   repository.INotificationRepository
}

func NewNotificationSvc(logger logger.ILogger, repo repository.INotificationRepository) INotificationSvc {
	return &NotificationSvc{
		logger: logger,
		repo:   repo,
	}
}

func (s *NotificationSvc) SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	return nil, errorx.New(errorx.ErrInternal, "Method unimplemented")
}
