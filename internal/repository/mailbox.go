package repository

import (
	"context"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
	"gorm.io/gorm"
)

type IMailboxRepository interface {
	IRepository[model.Mailbox]
	ListByCreatedBy(ctx context.Context, createdBy string, limit int) ([]model.Mailbox, error)
	FindOneByIdAndCreatedBy(ctx context.Context, id, createdBy string) *model.Mailbox
	MarkRead(ctx context.Context, id, createdBy string, readAt time.Time) error
}

type mailboxRepository struct {
	Repository[model.Mailbox]
}

func NewMailboxRepository(dbClient *gorm.DB) IMailboxRepository {
	return &mailboxRepository{Repository: Repository[model.Mailbox]{dbClient: dbClient}}
}

func (r *mailboxRepository) ListByCreatedBy(ctx context.Context, createdBy string, limit int) ([]model.Mailbox, error) {
	if createdBy == "" {
		return nil, nil
	}
	q := r.dbClient.WithContext(ctx).
		Where("created_by = ?", createdBy).
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []model.Mailbox
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *mailboxRepository) FindOneByIdAndCreatedBy(ctx context.Context, id, createdBy string) *model.Mailbox {
	if id == "" || createdBy == "" {
		return nil
	}
	var row model.Mailbox
	if err := r.dbClient.WithContext(ctx).
		Where("id = ? AND created_by = ?", id, createdBy).
		First(&row).Error; err != nil {
		return nil
	}
	return &row
}

func (r *mailboxRepository) MarkRead(ctx context.Context, id, createdBy string, readAt time.Time) error {
	if id == "" || createdBy == "" {
		return gorm.ErrRecordNotFound
	}
	res := r.dbClient.WithContext(ctx).Model(&model.Mailbox{}).
		Where("id = ? AND created_by = ?", id, createdBy).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": readAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
