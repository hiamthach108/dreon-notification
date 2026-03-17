package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var _ repository.INotificationRepository = (*mockNotificationRepo)(nil)

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
