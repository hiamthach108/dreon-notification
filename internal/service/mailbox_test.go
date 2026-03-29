package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openSQLiteMailboxSvc(t *testing.T) (IMailboxSvc, *gorm.DB) {
	t.Helper()
	memName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + memName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Notification{}, &model.Mailbox{}))
	repo := repository.NewMailboxRepository(db)
	return NewMailboxSvc(repo), db
}

func insertTestNotification(t *testing.T, db *gorm.DB, idem string) string {
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
	return n.ID
}

func TestMailboxSvc_Create_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteMailboxSvc(t)
	notifID := insertTestNotification(t, db, "idem-svc-create-1")
	userID := "11111111-1111-4111-8111-111111111111"

	got, err := svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         userID,
		Title:          "  Hello  ",
		Message:        "Body",
		Group:          " alerts ",
		NotificationID: notifID,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, "Hello", got.Title)
	assert.Equal(t, "Body", got.Message)
	assert.Equal(t, "alerts", got.Group)
	assert.Equal(t, userID, got.CreatedBy)
	assert.Equal(t, notifID, got.NotificationID)
	assert.False(t, got.IsRead)
}

func TestMailboxSvc_Create_ValidationError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteMailboxSvc(t)
	notifID := insertTestNotification(t, db, "idem-svc-val-1")

	_, err := svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         "",
		Title:          "T",
		NotificationID: notifID,
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	_, err = svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         "not-a-uuid",
		Title:          "T",
		NotificationID: notifID,
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))
}

func TestMailboxSvc_ListForUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteMailboxSvc(t)
	notifID := insertTestNotification(t, db, "idem-svc-list-1")
	userID := "22222222-2222-4222-8222-222222222222"

	list, err := svc.ListForUser(ctx, userID, 0)
	require.NoError(t, err)
	assert.Empty(t, list)

	_, err = svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         userID,
		Title:          "First",
		NotificationID: notifID,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         userID,
		Title:          "Second",
		NotificationID: notifID,
	})
	require.NoError(t, err)

	list, err = svc.ListForUser(ctx, userID, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "Second", list[0].Title)
	assert.Equal(t, "First", list[1].Title)

	list, err = svc.ListForUser(ctx, userID, 1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Second", list[0].Title)
}

func TestMailboxSvc_ListForUser_EmptyUserID(t *testing.T) {
	t.Parallel()
	svc, _ := openSQLiteMailboxSvc(t)
	_, err := svc.ListForUser(context.Background(), "", 0)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))
}

func TestMailboxSvc_GetForUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteMailboxSvc(t)
	notifID := insertTestNotification(t, db, "idem-svc-get-1")
	userA := "33333333-3333-4333-8333-333333333333"
	userB := "44444444-4444-4444-8444-444444444444"

	created, err := svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         userA,
		Title:          "Mine",
		NotificationID: notifID,
	})
	require.NoError(t, err)

	_, err = svc.GetForUser(ctx, "", userA)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	_, err = svc.GetForUser(ctx, created.ID, userB)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrNotFound, errorx.GetCode(err))

	got, err := svc.GetForUser(ctx, created.ID, userA)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "Mine", got.Title)
}

func TestMailboxSvc_MarkAsRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteMailboxSvc(t)
	notifID := insertTestNotification(t, db, "idem-svc-read-1")
	userID := "55555555-5555-4555-8555-555555555555"

	created, err := svc.Create(ctx, &aggregate.CreateMailboxReq{
		UserID:         userID,
		Title:          "Unread",
		NotificationID: notifID,
	})
	require.NoError(t, err)
	assert.False(t, created.IsRead)

	err = svc.MarkAsRead(ctx, "", userID)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	err = svc.MarkAsRead(ctx, created.ID, "66666666-6666-4666-8666-666666666666")
	require.Error(t, err)
	assert.Equal(t, errorx.ErrNotFound, errorx.GetCode(err))

	err = svc.MarkAsRead(ctx, created.ID, userID)
	require.NoError(t, err)

	got, err := svc.GetForUser(ctx, created.ID, userID)
	require.NoError(t, err)
	assert.True(t, got.IsRead)
	require.NotNil(t, got.ReadAt)
	assert.False(t, got.ReadAt.Before(time.Now().UTC().Add(-time.Minute)))
}
