package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type INotificationRepository interface {
	IRepository[model.Notification]
	RunInTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	LockPendingRetriesDueForUpdate(tx *gorm.DB, limit int, notAfter time.Time) ([]model.Notification, error)
	UpdateNextRetryAt(tx *gorm.DB, id string, at time.Time) error
	RecordSendFailure(ctx context.Context, id string, backoffInitialSec, backoffMaxSec int) error
	FindDueScheduledNotifications(ctx context.Context, limit int, notAfter time.Time) ([]model.Notification, error)
}

type notificationRepository struct {
	Repository[model.Notification]
}

func NewNotificationRepository(dbClient *gorm.DB) INotificationRepository {
	return &notificationRepository{Repository: Repository[model.Notification]{dbClient: dbClient}}
}

func (r *notificationRepository) RunInTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.dbClient.WithContext(ctx).Transaction(fn)
}

func (r *notificationRepository) FindDueScheduledNotifications(ctx context.Context, limit int, notAfter time.Time) ([]model.Notification, error) {
	if limit <= 0 {
		return nil, nil
	}
	var zero time.Time
	var rows []model.Notification
	err := r.dbClient.WithContext(ctx).
		Where(`status = ? AND attempt_count = 0 AND scheduled_at > ? AND scheduled_at <= ?`,
			model.NotificationStatusPending, zero, notAfter).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *notificationRepository) LockPendingRetriesDueForUpdate(tx *gorm.DB, limit int, notAfter time.Time) ([]model.Notification, error) {
	if limit <= 0 {
		return nil, nil
	}
	var rows []model.Notification
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(`status = ? AND attempt_count >= 1 AND attempt_count < max_attempts
			AND (next_retry_at IS NULL OR next_retry_at <= ?)`,
			model.NotificationStatusPending, notAfter).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *notificationRepository) UpdateNextRetryAt(tx *gorm.DB, id string, at time.Time) error {
	return tx.Model(&model.Notification{}).Where("id = ?", id).Update("next_retry_at", at).Error
}

func (r *notificationRepository) RecordSendFailure(ctx context.Context, id string, backoffInitialSec, backoffMaxSec int) error {
	if backoffInitialSec <= 0 {
		backoffInitialSec = 1
	}
	if backoffMaxSec <= 0 {
		backoffMaxSec = backoffInitialSec
	}
	if backoffMaxSec < backoffInitialSec {
		backoffMaxSec = backoffInitialSec
	}
	db := r.dbClient.WithContext(ctx)
	var stmt gorm.Statement
	stmt.DB = db
	if err := stmt.Parse(&model.Notification{}); err != nil {
		return fmt.Errorf("parse notification schema: %w", err)
	}
	table := stmt.Schema.Table
	if table == "" {
		table = "notifications"
	}
	q := fmt.Sprintf(`
		UPDATE %s
		SET attempt_count = attempt_count + 1,
		    status = CASE WHEN attempt_count + 1 >= max_attempts THEN ? ELSE status END,
		    next_retry_at = CASE
		      WHEN attempt_count + 1 >= max_attempts THEN next_retry_at
		      ELSE NOW() + (
		        LEAST(
		          (?::double precision * POWER(2::double precision, attempt_count::double precision))::bigint,
		          ?::bigint
		        ) * INTERVAL '1 second'
		      )
		    END
		WHERE id = ?`, table)
	return db.Exec(q,
		string(model.NotificationStatusFailed),
		backoffInitialSec,
		backoffMaxSec,
		id,
	).Error
}
