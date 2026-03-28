package repository

import (
	"context"

	"github.com/hiamthach108/dreon-notification/internal/model"
	"gorm.io/gorm"
)

type IPushTopicRepository interface {
	IRepository[model.PushTopic]
	FindByName(ctx context.Context, name string) *model.PushTopic
}

type pushTopicRepository struct {
	Repository[model.PushTopic]
}

func NewPushTopicRepository(dbClient *gorm.DB) IPushTopicRepository {
	return &pushTopicRepository{Repository: Repository[model.PushTopic]{dbClient: dbClient}}
}

func (r *pushTopicRepository) FindByName(ctx context.Context, name string) *model.PushTopic {
	var result model.PushTopic
	if err := r.dbClient.WithContext(ctx).Where("name = ?", name).First(&result).Error; err != nil {
		return nil
	}
	return &result
}
