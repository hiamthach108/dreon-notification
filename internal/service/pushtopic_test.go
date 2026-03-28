package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openSQLitePushTopicRepo(t *testing.T) repository.IPushTopicRepository {
	t.Helper()
	memName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + memName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PushTopic{}))
	return repository.NewPushTopicRepository(db)
}

func TestPushTopicSvc_GetAll_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := openSQLitePushTopicRepo(t)
	svc := NewPushTopicSvc(repo)

	list, err := svc.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 0)

	created, err := svc.Create(ctx, &aggregate.CreatePushTopicReq{Name: "group-alpha", Description: "test"})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "group-alpha", created.Name)
	assert.True(t, created.IsActive)

	list, err = svc.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "group-alpha", list[0].Name)
}

func TestPushTopicSvc_Create_DuplicateName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := NewPushTopicSvc(openSQLitePushTopicRepo(t))

	_, err := svc.Create(ctx, &aggregate.CreatePushTopicReq{Name: "dup"})
	require.NoError(t, err)
	_, err = svc.Create(ctx, &aggregate.CreatePushTopicReq{Name: "dup"})
	require.Error(t, err)
}

func TestPushTopicSvc_Update(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := NewPushTopicSvc(openSQLitePushTopicRepo(t))

	created, err := svc.Create(ctx, &aggregate.CreatePushTopicReq{Name: "orig", Description: "d0"})
	require.NoError(t, err)

	falseVal := false
	name := "renamed"
	err = svc.Update(ctx, created.ID, &aggregate.UpdatePushTopicReq{
		Name:        &name,
		Description: strPtr("d1"),
		IsActive:    &falseVal,
	})
	require.NoError(t, err)

	list, err := svc.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "renamed", list[0].Name)
	assert.Equal(t, "d1", list[0].Description)
	assert.False(t, list[0].IsActive)
}

func TestPushTopicSvc_Update_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := NewPushTopicSvc(openSQLitePushTopicRepo(t))
	name := "x"
	err := svc.Update(ctx, "00000000-0000-0000-0000-000000000000", &aggregate.UpdatePushTopicReq{Name: &name})
	require.Error(t, err)
}

func TestPushTopicSvc_Update_NoFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := NewPushTopicSvc(openSQLitePushTopicRepo(t))
	created, err := svc.Create(ctx, &aggregate.CreatePushTopicReq{Name: "only"})
	require.NoError(t, err)

	err = svc.Update(ctx, created.ID, &aggregate.UpdatePushTopicReq{})
	require.Error(t, err)
}

func strPtr(s string) *string { return &s }
