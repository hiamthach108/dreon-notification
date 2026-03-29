package repository

import (
	"context"

	"github.com/hiamthach108/dreon-notification/internal/model"
	"gorm.io/gorm"
)

type IUserFCMTokenRepository interface {
	IRepository[model.UserFCMToken]
	FindByToken(ctx context.Context, token string) *model.UserFCMToken
	ListByCreatedBy(ctx context.Context, createdBy string) ([]model.UserFCMToken, error)
	DeleteByIdAndCreatedBy(ctx context.Context, id, createdBy string) error
}

type userFCMTokenRepository struct {
	Repository[model.UserFCMToken]
}

func NewUserFCMTokenRepository(dbClient *gorm.DB) IUserFCMTokenRepository {
	return &userFCMTokenRepository{Repository: Repository[model.UserFCMToken]{dbClient: dbClient}}
}

func (r *userFCMTokenRepository) FindByToken(ctx context.Context, token string) *model.UserFCMToken {
	if token == "" {
		return nil
	}
	var row model.UserFCMToken
	if err := r.dbClient.WithContext(ctx).Where("token = ?", token).First(&row).Error; err != nil {
		return nil
	}
	return &row
}

func (r *userFCMTokenRepository) ListByCreatedBy(ctx context.Context, createdBy string) ([]model.UserFCMToken, error) {
	if createdBy == "" {
		return nil, nil
	}
	var rows []model.UserFCMToken
	err := r.dbClient.WithContext(ctx).
		Where("created_by = ?", createdBy).
		Order("updated_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *userFCMTokenRepository) DeleteByIdAndCreatedBy(ctx context.Context, id, createdBy string) error {
	if id == "" || createdBy == "" {
		return gorm.ErrRecordNotFound
	}
	res := r.dbClient.WithContext(ctx).
		Where("id = ? AND created_by = ?", id, createdBy).
		Delete(&model.UserFCMToken{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
