package repository

import (
	"github.com/hiamthach108/dreon-notification/internal/model"
	"gorm.io/gorm"
)

type INotificationRepository interface {
	IRepository[model.Notification]
}

type notificationRepository struct {
	Repository[model.Notification]
}

func NewNotificationRepository(dbClient *gorm.DB) INotificationRepository {
	return &notificationRepository{Repository: Repository[model.Notification]{dbClient: dbClient}}
}
