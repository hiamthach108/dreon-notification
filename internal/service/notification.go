package service

import (
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
)

type INotificationSvc interface {
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
