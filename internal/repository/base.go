package repository

import (
	"context"

	"gorm.io/gorm"
)

type IRepository[T any] interface {
	FindAll(ctx context.Context) ([]T, error)
	FindOneById(ctx context.Context, id string) *T
	FindByIds(ctx context.Context, ids []string) ([]T, error)
	Create(ctx context.Context, model *T) (*T, error)
	BulkCreate(ctx context.Context, inputs []T) error
	Update(ctx context.Context, id string, value T, field ...string) error
	DeleteById(ctx context.Context, id string) error
}

type Repository[T any] struct {
	dbClient *gorm.DB
}

var _ IRepository[any] = &Repository[any]{}

func (r *Repository[T]) FindAll(ctx context.Context) ([]T, error) {
	var results []T
	if err := r.dbClient.WithContext(ctx).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *Repository[T]) FindOneById(ctx context.Context, id string) *T {
	var result T
	if err := r.dbClient.WithContext(ctx).First(&result, "id = ?", id).Error; err != nil {
		return nil
	}
	return &result
}

func (r *Repository[T]) FindByIds(ctx context.Context, ids []string) ([]T, error) {
	var results []T
	if err := r.dbClient.WithContext(ctx).Find(&results, "id IN (?)", ids).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *Repository[T]) Create(ctx context.Context, model *T) (*T, error) {
	if err := r.dbClient.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return model, nil
}

func (r *Repository[T]) BulkCreate(ctx context.Context, inputs []T) error {
	if err := r.dbClient.WithContext(ctx).Create(&inputs).Error; err != nil {
		return err
	}
	return nil
}

func (r *Repository[T]) Update(ctx context.Context, id string, value T, field ...string) error {
	if err := r.dbClient.WithContext(ctx).Model(&value).Where("id = ?", id).Select(field).Updates(value).Error; err != nil {
		return err
	}
	return nil
}

func (r *Repository[T]) DeleteById(ctx context.Context, id string) error {
	if err := r.dbClient.WithContext(ctx).Delete(new(T), "id = ?", id).Error; err != nil {
		return err
	}
	return nil
}
