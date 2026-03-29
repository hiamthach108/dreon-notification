package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openSQLiteUserFCMTokenSvc(t *testing.T) (IUserFCMTokenSvc, *gorm.DB) {
	t.Helper()
	memName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + memName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserFCMToken{}))
	repo := repository.NewUserFCMTokenRepository(db)
	return NewUserFCMTokenSvc(repo), db
}

const (
	testFCMUserA = "11111111-1111-4111-8111-111111111111"
	testFCMUserB = "22222222-2222-4222-8222-222222222222"
)

func testFCMToken(suffix string) string {
	// min length 10 for RegisterUserFCMTokenReq validation
	return "fcm-token-" + suffix + "-0123456789"
}

func TestUserFCMTokenSvc_Register_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteUserFCMTokenSvc(t)
	tok := testFCMToken("create-1")

	got, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    tok,
		Platform: "IOS",
		DeviceMetadata: map[string]any{
			"model": "iPhone",
			"os":    "17",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, testFCMUserA, got.UserID)
	assert.Equal(t, tok, got.Token)
	assert.Equal(t, "IOS", got.Platform)
	assert.Equal(t, "iPhone", got.DeviceMetadata["model"])
	assert.Equal(t, "17", got.DeviceMetadata["os"])

	var count int64
	require.NoError(t, db.Model(&model.UserFCMToken{}).Where("token = ?", tok).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestUserFCMTokenSvc_Register_UpsertSameToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteUserFCMTokenSvc(t)
	tok := testFCMToken("upsert-1")

	first, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    tok,
		Platform: "IOS",
		DeviceMetadata: map[string]any{
			"v": float64(1),
		},
	})
	require.NoError(t, err)
	id := first.ID

	second, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserB,
		Token:    tok,
		Platform: "ANDROID",
		DeviceMetadata: map[string]any{
			"v": float64(2),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, id, second.ID, "same token should update the same row")
	assert.Equal(t, testFCMUserB, second.UserID)
	assert.Equal(t, "ANDROID", second.Platform)
	assert.Equal(t, float64(2), second.DeviceMetadata["v"])

	var count int64
	require.NoError(t, db.Model(&model.UserFCMToken{}).Where("token = ?", tok).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestUserFCMTokenSvc_Register_ValidationError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, _ := openSQLiteUserFCMTokenSvc(t)
	tok := testFCMToken("val")

	_, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   "",
		Token:    tok,
		Platform: "IOS",
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	_, err = svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   "not-uuid",
		Token:    tok,
		Platform: "IOS",
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	_, err = svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    "short",
		Platform: "IOS",
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	_, err = svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    tok,
		Platform: "DESKTOP",
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))
}

func TestUserFCMTokenSvc_Register_TokenWhitespaceOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, _ := openSQLiteUserFCMTokenSvc(t)

	_, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    "          ",
		Platform: "WEB",
	})
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))
}

func TestUserFCMTokenSvc_ListForUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, _ := openSQLiteUserFCMTokenSvc(t)

	list, err := svc.ListForUser(ctx, testFCMUserA)
	require.NoError(t, err)
	assert.Empty(t, list)

	_, err = svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    testFCMToken("list-a"),
		Platform: "IOS",
	})
	require.NoError(t, err)
	_, err = svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    testFCMToken("list-b"),
		Platform: "WEB",
	})
	require.NoError(t, err)

	list, err = svc.ListForUser(ctx, testFCMUserA)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, testFCMUserA, list[0].UserID)
	assert.Equal(t, testFCMUserA, list[1].UserID)

	other, err := svc.ListForUser(ctx, testFCMUserB)
	require.NoError(t, err)
	assert.Empty(t, other)
}

func TestUserFCMTokenSvc_ListForUser_EmptyUserID(t *testing.T) {
	t.Parallel()
	svc, _ := openSQLiteUserFCMTokenSvc(t)
	_, err := svc.ListForUser(context.Background(), "")
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))
}

func TestUserFCMTokenSvc_DeleteForUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, db := openSQLiteUserFCMTokenSvc(t)

	got, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    testFCMToken("del-1"),
		Platform: "IOS",
	})
	require.NoError(t, err)

	err = svc.DeleteForUser(ctx, "", testFCMUserA)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrBadRequest, errorx.GetCode(err))

	err = svc.DeleteForUser(ctx, got.ID, testFCMUserB)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrNotFound, errorx.GetCode(err))

	err = svc.DeleteForUser(ctx, got.ID, testFCMUserA)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&model.UserFCMToken{}).Where("id = ?", got.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)

	err = svc.DeleteForUser(ctx, got.ID, testFCMUserA)
	require.Error(t, err)
	assert.Equal(t, errorx.ErrNotFound, errorx.GetCode(err))
}

func TestUserFCMTokenSvc_Register_WithoutDeviceMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, _ := openSQLiteUserFCMTokenSvc(t)

	got, err := svc.Register(ctx, &aggregate.RegisterUserFCMTokenReq{
		UserID:   testFCMUserA,
		Token:    testFCMToken("nometa"),
		Platform: "WEB",
	})
	require.NoError(t, err)
	assert.Nil(t, got.DeviceMetadata)
}
