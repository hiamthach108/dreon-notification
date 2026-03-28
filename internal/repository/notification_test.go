package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationRepository_FindDueScheduledNotifications(t *testing.T) {
	t.Parallel()
	memName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + memName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Notification{}))

	repo := NewNotificationRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	paramsJSON, err := json.Marshal(map[string]any{"k": "v"})
	require.NoError(t, err)

	base := model.Notification{
		IdempotencyKey: "idem-past",
		Source:         "src",
		Channel:        model.NotificationChannelEmail,
		Type:           model.NotificationTypeWelcome,
		Status:         model.NotificationStatusPending,
		Title:          "T",
		Recipients:     []string{"a@b.com"},
		Params:         paramsJSON,
		Provider:       model.NotificationProviderResend,
		MaxAttempts:    3,
		AttemptCount:   0,
		ScheduledAt:    past,
	}
	require.NoError(t, db.Create(&base).Error)

	immediate := base
	immediate.ID = ""
	immediate.IdempotencyKey = "idem-immediate"
	immediate.ScheduledAt = time.Time{}
	require.NoError(t, db.Create(&immediate).Error)

	futureRow := base
	futureRow.ID = ""
	futureRow.IdempotencyKey = "idem-future"
	futureRow.ScheduledAt = future
	require.NoError(t, db.Create(&futureRow).Error)

	failedAttempt := base
	failedAttempt.ID = ""
	failedAttempt.IdempotencyKey = "idem-retry"
	failedAttempt.ScheduledAt = past
	failedAttempt.AttemptCount = 1
	require.NoError(t, db.Create(&failedAttempt).Error)

	rows, err := repo.FindDueScheduledNotifications(ctx, 10, now)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "idem-past", rows[0].IdempotencyKey)

	empty, err := repo.FindDueScheduledNotifications(ctx, 0, now)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
