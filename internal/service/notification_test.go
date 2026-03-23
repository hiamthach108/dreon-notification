package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/internal/shared/constant"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// templateDirForTest returns path to templates/email relative to the module root (run from repo root or internal/service).
func templateDirForTest(t *testing.T) string {
	t.Helper()
	// When tests run, cwd is often the package dir (internal/service); go up to repo root.
	dir := filepath.Join("..", "..", "templates", "email")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	// When run from repo root (e.g. go test ./internal/service -run ... from root).
	return filepath.Join("templates", "email")
}

// mockEmailClient captures the last SendEmail call for assertions.
type mockEmailClient struct {
	last *email.EmailData
	err  error
}

func (m *mockEmailClient) SendEmail(ctx context.Context, data *email.EmailData) error {
	m.last = data
	return m.err
}

type mockMessagePublisher struct {
	lastTopic string
	messages  []*message.Message
	err       error
}

func (m *mockMessagePublisher) Publish(topic string, msgs ...*message.Message) error {
	m.lastTopic = topic
	m.messages = append(m.messages, msgs...)
	return m.err
}

var _ notificationMessagePublisher = (*mockMessagePublisher)(nil)

// mockNotificationRepo is a no-op repo for tests that don't persist.
type mockNotificationRepo struct{}

func (mockNotificationRepo) FindAll(ctx context.Context) ([]model.Notification, error) {
	return nil, nil
}
func (mockNotificationRepo) FindOneById(ctx context.Context, id string) *model.Notification {
	return nil
}
func (mockNotificationRepo) FindByIds(ctx context.Context, ids []string) ([]model.Notification, error) {
	return nil, nil
}
func (mockNotificationRepo) Create(ctx context.Context, n *model.Notification) (*model.Notification, error) {
	return n, nil
}
func (mockNotificationRepo) BulkCreate(ctx context.Context, inputs []model.Notification) error {
	return nil
}
func (mockNotificationRepo) Update(ctx context.Context, id string, value model.Notification, field ...string) error {
	return nil
}
func (mockNotificationRepo) DeleteById(ctx context.Context, id string) error {
	return nil
}
func (mockNotificationRepo) RunInTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return nil
}
func (mockNotificationRepo) LockPendingRetriesDueForUpdate(tx *gorm.DB, limit int, notAfter time.Time) ([]model.Notification, error) {
	return nil, nil
}
func (mockNotificationRepo) UpdateNextRetryAt(tx *gorm.DB, id string, at time.Time) error {
	return nil
}
func (mockNotificationRepo) RecordSendFailure(ctx context.Context, id string, initial, max int) error {
	return nil
}

var _ repository.INotificationRepository = (*mockNotificationRepo)(nil)

// captureNotificationRepo records RecordSendFailure, FindOneById, and Update for assertions.
type captureNotificationRepo struct {
	mockNotificationRepo
	findOne          *model.Notification
	recordFailureIDs []string
	lastUpdateID     string
	lastUpdate       model.Notification
}

func (c *captureNotificationRepo) FindOneById(ctx context.Context, id string) *model.Notification {
	if c.findOne == nil || c.findOne.ID != id {
		return nil
	}
	return c.findOne
}

func (c *captureNotificationRepo) RecordSendFailure(ctx context.Context, id string, initial, max int) error {
	c.recordFailureIDs = append(c.recordFailureIDs, id)
	return nil
}

func (c *captureNotificationRepo) Update(ctx context.Context, id string, value model.Notification, field ...string) error {
	c.lastUpdateID = id
	c.lastUpdate = value
	return nil
}

func samplePendingEmailNotification(t *testing.T) model.Notification {
	t.Helper()
	paramsJSON, err := json.Marshal(map[string]any{"Name": "Bob", "AppURL": "https://app.example.com"})
	require.NoError(t, err)
	return model.Notification{
		BaseModel:      model.BaseModel{ID: "notif-retry-1"},
		IdempotencyKey: "idem-1",
		Source:         "test",
		Channel:        model.NotificationChannelEmail,
		Type:           model.NotificationTypeWelcome,
		Status:         model.NotificationStatusPending,
		Title:          "Welcome",
		Recipients:     []string{"user@example.com"},
		Params:         paramsJSON,
		Provider:       model.NotificationProviderResend,
		MaxAttempts:    3,
		AttemptCount:   1,
	}
}

func TestNotificationSvc_SendEmail_Welcome(t *testing.T) {
	ctx := context.Background()
	cfg := &config.AppConfig{}
	cfg.Email.Sender = "noreply@example.com"
	cfg.Email.TemplateDir = templateDirForTest(t)

	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	renderer := email.NewRenderer(cfg)
	mockClient := &mockEmailClient{}

	svc := NewNotificationSvc(
		appLogger,
		&mockNotificationRepo{},
		nil, // publisher not needed for SendNotification tests
		mockClient,
		renderer,
		sms.NewMockClient(),
		nil,
		cfg,
	)

	req := &aggregate.SendNotificationReq{
		Channel:    string(model.NotificationChannelEmail),
		Type:       string(model.NotificationTypeWelcome),
		Title:      "Welcome!",
		Recipients: []string{"user@example.com"},
		Params: map[string]any{
			"Name":   "Alice",
			"AppURL": "https://app.example.com",
		},
	}

	resp, err := svc.SendNotification(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.NotNil(t, mockClient.last, "SendEmail should have been called")
	assert.Equal(t, "noreply@example.com", mockClient.last.From)
	assert.Equal(t, []string{"user@example.com"}, mockClient.last.To)
	assert.Equal(t, "Welcome!", mockClient.last.Subject)
	assert.Contains(t, mockClient.last.HTML, "Alice")
	assert.Contains(t, mockClient.last.HTML, "https://app.example.com")
}

func TestNotificationSvc_SendEmail_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	cfg := &config.AppConfig{}
	cfg.Email.Sender = "noreply@example.com"
	cfg.Email.TemplateDir = templateDirForTest(t)

	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	renderer := email.NewRenderer(cfg)
	mockClient := &mockEmailClient{}

	svc := NewNotificationSvc(
		appLogger,
		&mockNotificationRepo{},
		nil, // publisher not needed for SendNotification tests
		mockClient,
		renderer,
		sms.NewMockClient(),
		nil,
		cfg,
	)

	req := &aggregate.SendNotificationReq{
		Channel:    string(model.NotificationChannelEmail),
		Type:       "UNKNOWN_TYPE",
		Title:      "Test",
		Recipients: []string{"user@example.com"},
	}

	_, err = svc.SendNotification(ctx, req)
	require.Error(t, err)
	assert.Nil(t, mockClient.last)
}

func TestNotificationSvc_SendNotification_ChannelSwitch(t *testing.T) {
	ctx := context.Background()
	cfg := &config.AppConfig{}
	cfg.Email.TemplateDir = templateDirForTest(t)
	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	svc := NewNotificationSvc(
		appLogger,
		&mockNotificationRepo{},
		nil, // publisher not needed for SendNotification tests
		&mockEmailClient{},
		email.NewRenderer(cfg),
		sms.NewMockClient(),
		nil,
		cfg,
	)

	t.Run("SMS returns error when client not configured", func(t *testing.T) {
		_, err := svc.SendNotification(ctx, &aggregate.SendNotificationReq{
			Channel:    string(model.NotificationChannelSms),
			Type:       string(model.NotificationTypeVerifyOTP),
			Title:      "OTP",
			Message:    "Your code is 123456",
			Recipients: []string{"+1234567890"},
		})
		require.Error(t, err)
		// MockClient returns "not configured" when SendSMS is called
		assert.Contains(t, err.Error(), "not configured")
	})

	t.Run("PUSH not implemented", func(t *testing.T) {
		_, err := svc.SendNotification(ctx, &aggregate.SendNotificationReq{
			Channel:    string(model.NotificationChannelPush),
			Title:      "Push",
			Recipients: []string{"device-token"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("IN_APP not implemented", func(t *testing.T) {
		_, err := svc.SendNotification(ctx, &aggregate.SendNotificationReq{
			Channel:    string(model.NotificationChannelInApp),
			Title:      "In-app",
			Recipients: []string{"user-id"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

func TestNotificationSvc_EnqueuePendingRetries_PublishesToRetryTopic(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Notification{}))

	past := time.Now().UTC().Add(-time.Hour)
	paramsJSON, err := json.Marshal(map[string]any{"Name": "Bob"})
	require.NoError(t, err)
	n := model.Notification{
		BaseModel:      model.BaseModel{ID: "notif-retry-1"},
		IdempotencyKey: "idem-enqueue-1",
		Source:         "test",
		Channel:        model.NotificationChannelEmail,
		Type:           model.NotificationTypeWelcome,
		Status:         model.NotificationStatusPending,
		Title:          "Welcome",
		Recipients:     []string{"user@example.com"},
		Params:         paramsJSON,
		Provider:       model.NotificationProviderResend,
		MaxAttempts:    3,
		AttemptCount:   1,
		NextRetryAt:    &past,
	}
	require.NoError(t, db.Create(&n).Error)

	cfg := &config.AppConfig{}
	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	repo := repository.NewNotificationRepository(db)
	pub := &mockMessagePublisher{}
	svc := NewNotificationSvc(
		appLogger,
		repo,
		pub,
		&mockEmailClient{},
		email.NewRenderer(cfg),
		sms.NewMockClient(),
		nil,
		cfg,
	)

	require.NoError(t, svc.EnqueuePendingRetries(ctx, 10))
	require.Len(t, pub.messages, 1)
	assert.Equal(t, constant.EventTopicNotificationsRetry, pub.lastTopic)
	var p aggregate.NotificationRetryPayload
	require.NoError(t, json.Unmarshal(pub.messages[0].Payload, &p))
	assert.Equal(t, "notif-retry-1", p.NotificationID)
}

func TestNotificationSvc_EnqueuePendingRetries_ErrsWithoutPublisher(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Notification{}))

	cfg := &config.AppConfig{}
	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	repo := repository.NewNotificationRepository(db)
	svc := NewNotificationSvc(appLogger, repo, nil, &mockEmailClient{}, email.NewRenderer(cfg), sms.NewMockClient(), nil, cfg)
	require.Error(t, svc.EnqueuePendingRetries(ctx, 10))
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_Success(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.Email.Sender = "noreply@example.com"
	cfg.Email.TemplateDir = templateDirForTest(t)

	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	n := samplePendingEmailNotification(t)
	repo := &captureNotificationRepo{findOne: &n}
	svc := NewNotificationSvc(
		appLogger,
		repo,
		&mockMessagePublisher{},
		&mockEmailClient{},
		email.NewRenderer(cfg),
		sms.NewMockClient(),
		nil,
		cfg,
	)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{NotificationID: n.ID})
	require.NoError(t, err)
	msg := message.NewMessage("r-1", body)

	require.NoError(t, svc.ProcessNotificationRetryFromQueue(msg))
	assert.Equal(t, n.ID, repo.lastUpdateID)
	assert.Equal(t, model.NotificationStatusCompleted, repo.lastUpdate.Status)
	assert.False(t, repo.lastUpdate.SentAt.IsZero())
	assert.Empty(t, repo.recordFailureIDs)
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_RecordsFailureOnSendError(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.Email.Sender = "noreply@example.com"
	cfg.Email.TemplateDir = templateDirForTest(t)

	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	n := samplePendingEmailNotification(t)
	repo := &captureNotificationRepo{findOne: &n}
	svc := NewNotificationSvc(
		appLogger,
		repo,
		&mockMessagePublisher{},
		&mockEmailClient{err: errors.New("smtp down")},
		email.NewRenderer(cfg),
		sms.NewMockClient(),
		nil,
		cfg,
	)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{NotificationID: n.ID})
	require.NoError(t, err)
	msg := message.NewMessage("r-2", body)

	require.NoError(t, svc.ProcessNotificationRetryFromQueue(msg))
	assert.Equal(t, []string{n.ID}, repo.recordFailureIDs)
	assert.Empty(t, repo.lastUpdateID)
}

func TestNotificationSvc_ProcessNotificationFromQueue_RecordsFailureOnSendError(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.Email.Sender = "noreply@example.com"
	cfg.Email.TemplateDir = templateDirForTest(t)

	appLogger, err := logger.NewLogger(cfg)
	require.NoError(t, err)

	repo := &captureNotificationRepo{}
	svc := NewNotificationSvc(
		appLogger,
		repo,
		nil,
		&mockEmailClient{err: errors.New("send failed")},
		email.NewRenderer(cfg),
		sms.NewMockClient(),
		nil,
		cfg,
	)

	payload := aggregate.NotificationEnqueuePayload{
		NotificationID: "q-notif-1",
		Req: aggregate.SendNotificationReq{
			IdempotencyKey: "k",
			Source:         "api",
			Channel:        string(model.NotificationChannelEmail),
			Type:           string(model.NotificationTypeWelcome),
			Title:          "T",
			Recipients:     []string{"a@b.com"},
			Params:         map[string]any{"Name": "X", "AppURL": "https://x.com"},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	msg := message.NewMessage("uuid-1", body)

	require.NoError(t, svc.ProcessNotificationFromQueue(msg))
	assert.Equal(t, []string{"q-notif-1"}, repo.recordFailureIDs)
}
