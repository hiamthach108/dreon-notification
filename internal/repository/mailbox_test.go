package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openSQLiteMailboxRepo(t *testing.T) (IMailboxRepository, *gorm.DB) {
	t.Helper()
	memName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + memName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Notification{}, &model.Mailbox{}))
	return NewMailboxRepository(db), db
}

func testNotificationRow(t *testing.T, db *gorm.DB, idem string) *model.Notification {
	t.Helper()
	paramsJSON, err := json.Marshal(map[string]any{"k": "v"})
	require.NoError(t, err)
	n := &model.Notification{
		IdempotencyKey: idem,
		Source:         "test",
		Channel:        model.NotificationChannelInApp,
		Type:           model.NotificationTypeWelcome,
		Status:         model.NotificationStatusPending,
		Title:          "N",
		Message:        "M",
		Recipients:     []string{"u"},
		Params:         paramsJSON,
		Provider:       model.NotificationProviderFirebase,
		MaxAttempts:    3,
	}
	require.NoError(t, db.Create(n).Error)
	return n
}

func TestMailboxRepository_ListByCreatedBy_EmptyCreatedBy(t *testing.T) {
	t.Parallel()
	repo, _ := openSQLiteMailboxRepo(t)
	ctx := context.Background()
	rows, err := repo.ListByCreatedBy(ctx, "", 10)
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestMailboxRepository_ListByCreatedBy_OrderAndLimit(t *testing.T) {
	t.Parallel()
	repo, db := openSQLiteMailboxRepo(t)
	ctx := context.Background()
	n := testNotificationRow(t, db, "idem-order-1")
	userID := "11111111-1111-4111-8111-111111111111"

	mb1 := model.Mailbox{Title: "older", Message: "a", NotificationID: n.ID}
	mb1.CreatedBy = userID
	require.NoError(t, db.Create(&mb1).Error)
	require.NoError(t, db.Model(&mb1).Update("created_at", time.Now().UTC().Add(-2*time.Hour)).Error)

	mb2 := model.Mailbox{Title: "newer", Message: "b", NotificationID: n.ID}
	mb2.CreatedBy = userID
	require.NoError(t, db.Create(&mb2).Error)

	all, err := repo.ListByCreatedBy(ctx, userID, 0)
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "newer", all[0].Title)
	assert.Equal(t, "older", all[1].Title)

	one, err := repo.ListByCreatedBy(ctx, userID, 1)
	require.NoError(t, err)
	require.Len(t, one, 1)
	assert.Equal(t, "newer", one[0].Title)
}

func TestMailboxRepository_FindOneByIdAndCreatedBy(t *testing.T) {
	t.Parallel()
	repo, db := openSQLiteMailboxRepo(t)
	ctx := context.Background()
	n := testNotificationRow(t, db, "idem-find-1")
	userA := "22222222-2222-4222-8222-222222222222"
	userB := "33333333-3333-4333-8333-333333333333"

	mb := model.Mailbox{Title: "x", NotificationID: n.ID}
	mb.CreatedBy = userA
	require.NoError(t, db.Create(&mb).Error)

	assert.Nil(t, repo.FindOneByIdAndCreatedBy(ctx, "", userA))
	assert.Nil(t, repo.FindOneByIdAndCreatedBy(ctx, mb.ID, ""))
	assert.Nil(t, repo.FindOneByIdAndCreatedBy(ctx, mb.ID, userB))

	got := repo.FindOneByIdAndCreatedBy(ctx, mb.ID, userA)
	require.NotNil(t, got)
	assert.Equal(t, mb.ID, got.ID)
	assert.Equal(t, userA, got.CreatedBy)
}

func TestMailboxRepository_MarkRead(t *testing.T) {
	t.Parallel()
	repo, db := openSQLiteMailboxRepo(t)
	ctx := context.Background()
	n := testNotificationRow(t, db, "idem-read-1")
	userID := "44444444-4444-4444-8444-444444444444"
	mb := model.Mailbox{Title: "r", NotificationID: n.ID}
	mb.CreatedBy = userID
	require.NoError(t, db.Create(&mb).Error)

	readAt := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	err := repo.MarkRead(ctx, mb.ID, userID, readAt)
	require.NoError(t, err)

	var stored model.Mailbox
	require.NoError(t, db.First(&stored, "id = ?", mb.ID).Error)
	assert.True(t, stored.IsRead)
	require.NotNil(t, stored.ReadAt)
	assert.True(t, readAt.Equal(*stored.ReadAt))

	err = repo.MarkRead(ctx, mb.ID, userID, readAt)
	require.NoError(t, err)

	err = repo.MarkRead(ctx, "00000000-0000-0000-0000-000000000000", userID, readAt)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	err = repo.MarkRead(ctx, mb.ID, "55555555-5555-4555-8555-555555555555", readAt)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	err = repo.MarkRead(ctx, "", userID, readAt)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}
