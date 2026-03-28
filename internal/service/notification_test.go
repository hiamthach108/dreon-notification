package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- test helpers ---

func templateDirForTest(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "templates", "email")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	return filepath.Join("templates", "email")
}

func testEmailConfig(t *testing.T) *config.AppConfig {
	t.Helper()
	cfg := &config.AppConfig{}
	cfg.Email.Sender = "noreply@example.com"
	cfg.Email.TemplateDir = templateDirForTest(t)
	return cfg
}

func testLogger(t *testing.T, cfg *config.AppConfig) logger.ILogger {
	t.Helper()
	log, err := logger.NewLogger(cfg)
	require.NoError(t, err)
	return log
}

func openSQLiteNotificationDB(t *testing.T) (*gorm.DB, repository.INotificationRepository) {
	t.Helper()
	// Unique in-memory DSN per test so t.Parallel() does not share one SQLite catalog.
	memName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + memName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Notification{}))
	return db, repository.NewNotificationRepository(db)
}

func newNotificationSvc(
	t *testing.T,
	repo repository.INotificationRepository,
	pub *amqp.Publisher,
	emailClient email.IEmailClient,
	cfg *config.AppConfig,
) INotificationSvc {
	t.Helper()
	if cfg == nil {
		cfg = testEmailConfig(t)
	}
	if emailClient == nil {
		emailClient = &mockEmailClient{}
	}
	return NewNotificationSvc(
		testLogger(t, cfg),
		repo,
		pub,
		emailClient,
		email.NewRenderer(cfg),
		sms.NewMockClient(),
		nil,
		cfg,
	)
}

func sampleEmailEnqueueReq() *aggregate.SendNotificationReq {
	return &aggregate.SendNotificationReq{
		IdempotencyKey: "idem-sample-1",
		Source:         "test",
		Channel:        string(model.NotificationChannelEmail),
		Type:           string(model.NotificationTypeWelcome),
		Title:          "Hello",
		Recipients:     []string{"user@example.com"},
		Params: map[string]any{
			"Name":   "Bob",
			"AppURL": "https://app.example.com",
		},
	}
}

// --- mocks ---

type mockEmailClient struct {
	last *email.EmailData
	err  error
}

func (m *mockEmailClient) SendEmail(ctx context.Context, data *email.EmailData) error {
	m.last = data
	return m.err
}

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

// stubCreateRepo assigns a fixed ID on Create when the model has none (enqueue tests).
type stubCreateRepo struct {
	mockNotificationRepo
	createdID string
}

func (s stubCreateRepo) Create(ctx context.Context, n *model.Notification) (*model.Notification, error) {
	if s.createdID != "" {
		n.ID = s.createdID
	}
	return n, nil
}

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

func samplePendingRetryNotification(t *testing.T) model.Notification {
	t.Helper()
	paramsJSON, err := json.Marshal(map[string]any{"Name": "Bob", "AppURL": "https://app.example.com"})
	require.NoError(t, err)
	return model.Notification{
		BaseModel:      model.BaseModel{ID: "notif-retry-1"},
		IdempotencyKey: "idem-retry-1",
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

// --- SendNotification ---

func TestNotificationSvc_SendNotification_Email_Welcome(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := testEmailConfig(t)
	mockClient := &mockEmailClient{}

	svc := newNotificationSvc(t, &mockNotificationRepo{}, nil, mockClient, cfg)

	req := sampleEmailEnqueueReq()
	req.Title = "Welcome!"
	req.Params = map[string]any{"Name": "Alice", "AppURL": "https://app.example.com"}

	resp, err := svc.SendNotification(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.NotNil(t, mockClient.last)
	assert.Equal(t, "noreply@example.com", mockClient.last.From)
	assert.Equal(t, []string{"user@example.com"}, mockClient.last.To)
	assert.Equal(t, "Welcome!", mockClient.last.Subject)
	assert.Contains(t, mockClient.last.HTML, "Alice")
	assert.Contains(t, mockClient.last.HTML, "https://app.example.com")
}

func TestNotificationSvc_SendNotification_Email_UnsupportedType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := testEmailConfig(t)
	mockClient := &mockEmailClient{}

	svc := newNotificationSvc(t, &mockNotificationRepo{}, nil, mockClient, cfg)

	_, err := svc.SendNotification(ctx, &aggregate.SendNotificationReq{
		Channel:    string(model.NotificationChannelEmail),
		Type:       "UNKNOWN_TYPE",
		Title:      "Test",
		Recipients: []string{"user@example.com"},
	})
	require.Error(t, err)
	assert.Nil(t, mockClient.last)
}

func TestNotificationSvc_SendNotification_ChannelRouting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := testEmailConfig(t)
	svc := newNotificationSvc(t, &mockNotificationRepo{}, nil, &mockEmailClient{}, cfg)

	tests := []struct {
		name    string
		req     *aggregate.SendNotificationReq
		wantErr string
	}{
		{
			name: "SMS_mock_not_configured",
			req: &aggregate.SendNotificationReq{
				Channel:    string(model.NotificationChannelSms),
				Type:       string(model.NotificationTypeVerifyOTP),
				Title:      "OTP",
				Message:    "Your code is 123456",
				Recipients: []string{"+1234567890"},
			},
			wantErr: "not configured",
		},
		{
			name: "PUSH_not_implemented",
			req: &aggregate.SendNotificationReq{
				Channel:    string(model.NotificationChannelPush),
				Title:      "Push",
				Recipients: []string{"device-token"},
			},
			wantErr: "not implemented",
		},
		{
			name: "IN_APP_not_implemented",
			req: &aggregate.SendNotificationReq{
				Channel:    string(model.NotificationChannelInApp),
				Title:      "In-app",
				Recipients: []string{"user-id"},
			},
			wantErr: "not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.SendNotification(ctx, tt.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// --- EnqueueNotification ---

func TestNotificationSvc_EnqueueNotification_ValidationError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := testEmailConfig(t)
	svc := newNotificationSvc(t, &mockNotificationRepo{}, nil, nil, cfg)

	req := sampleEmailEnqueueReq()
	req.Title = ""

	_, err := svc.EnqueueNotification(ctx, req)
	require.Error(t, err)
}

func TestNotificationSvc_EnqueueNotification_ErrWithoutPublisher(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := testEmailConfig(t)
	repo := stubCreateRepo{createdID: "nid-1"}
	svc := newNotificationSvc(t, repo, nil, nil, cfg)

	_, err := svc.EnqueueNotification(ctx, sampleEmailEnqueueReq())
	require.Error(t, err)
}

// --- EnqueuePendingRetries ---

func TestNotificationSvc_EnqueuePendingRetries_BatchSizeZero_NoOp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, repo := openSQLiteNotificationDB(t)
	svc := newNotificationSvc(t, repo, nil, nil, testEmailConfig(t))

	require.NoError(t, svc.EnqueuePendingRetries(ctx, 0))
}

func TestNotificationSvc_EnqueuePendingRetries_ErrWithoutPublisher(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, repo := openSQLiteNotificationDB(t)
	svc := newNotificationSvc(t, repo, nil, nil, testEmailConfig(t))

	require.Error(t, svc.EnqueuePendingRetries(ctx, 10))
}

// --- ProcessNotificationFromQueue ---

func TestNotificationSvc_ProcessNotificationFromQueue_InvalidPayload_NoPanic(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	svc := newNotificationSvc(t, &captureNotificationRepo{}, nil, &mockEmailClient{}, cfg)

	msg := message.NewMessage("m-1", []byte(`not json`))
	require.NoError(t, svc.ProcessNotificationFromQueue(msg))
}

func TestNotificationSvc_ProcessNotificationFromQueue_Success(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	repo := &captureNotificationRepo{}
	svc := newNotificationSvc(t, repo, nil, &mockEmailClient{}, cfg)

	payload := aggregate.NotificationEnqueuePayload{
		NotificationID: "q-ok-1",
		Req: func() aggregate.SendNotificationReq {
			r := *sampleEmailEnqueueReq()
			r.IdempotencyKey = "q-ok-idem"
			return r
		}(),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	msg := message.NewMessage("m-ok", body)

	require.NoError(t, svc.ProcessNotificationFromQueue(msg))
	assert.Equal(t, "q-ok-1", repo.lastUpdateID)
	assert.Equal(t, model.NotificationStatusCompleted, repo.lastUpdate.Status)
	assert.False(t, repo.lastUpdate.SentAt.IsZero())
	assert.Empty(t, repo.recordFailureIDs)
}

func TestNotificationSvc_ProcessNotificationFromQueue_SendFailure_RecordsFailure(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	repo := &captureNotificationRepo{}
	svc := newNotificationSvc(t, repo, nil, &mockEmailClient{err: errors.New("send failed")}, cfg)

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
	assert.Empty(t, repo.lastUpdateID)
}

// --- ProcessNotificationRetryFromQueue ---

func TestNotificationSvc_ProcessNotificationRetryFromQueue_Success(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	n := samplePendingRetryNotification(t)
	repo := &captureNotificationRepo{findOne: &n}
	svc := newNotificationSvc(t, repo, nil, &mockEmailClient{}, cfg)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{NotificationID: n.ID})
	require.NoError(t, err)
	msg := message.NewMessage("r-1", body)

	require.NoError(t, svc.ProcessNotificationRetryFromQueue(msg))
	assert.Equal(t, n.ID, repo.lastUpdateID)
	assert.Equal(t, model.NotificationStatusCompleted, repo.lastUpdate.Status)
	assert.False(t, repo.lastUpdate.SentAt.IsZero())
	assert.Empty(t, repo.recordFailureIDs)
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_SendFailure_RecordsFailure(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	n := samplePendingRetryNotification(t)
	repo := &captureNotificationRepo{findOne: &n}
	svc := newNotificationSvc(t, repo, nil, &mockEmailClient{err: errors.New("smtp down")}, cfg)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{NotificationID: n.ID})
	require.NoError(t, err)
	msg := message.NewMessage("r-2", body)

	require.NoError(t, svc.ProcessNotificationRetryFromQueue(msg))
	assert.Equal(t, []string{n.ID}, repo.recordFailureIDs)
	assert.Empty(t, repo.lastUpdateID)
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_InvalidPayload_NoPanic(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	svc := newNotificationSvc(t, &captureNotificationRepo{}, nil, &mockEmailClient{}, cfg)

	require.NoError(t, svc.ProcessNotificationRetryFromQueue(message.NewMessage("x", []byte(`{`))))
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_EmptyNotificationID_NoPanic(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	svc := newNotificationSvc(t, &captureNotificationRepo{}, nil, &mockEmailClient{}, cfg)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{})
	require.NoError(t, err)
	require.NoError(t, svc.ProcessNotificationRetryFromQueue(message.NewMessage("x", body)))
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_RowNotFound_NoPanic(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	svc := newNotificationSvc(t, &captureNotificationRepo{}, nil, &mockEmailClient{}, cfg)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{NotificationID: "missing"})
	require.NoError(t, err)
	require.NoError(t, svc.ProcessNotificationRetryFromQueue(message.NewMessage("x", body)))
}

func TestNotificationSvc_ProcessNotificationRetryFromQueue_NotPending_NoUpdate(t *testing.T) {
	t.Parallel()
	cfg := testEmailConfig(t)
	n := samplePendingRetryNotification(t)
	n.Status = model.NotificationStatusCompleted
	repo := &captureNotificationRepo{findOne: &n}
	svc := newNotificationSvc(t, repo, nil, &mockEmailClient{}, cfg)

	body, err := json.Marshal(aggregate.NotificationRetryPayload{NotificationID: n.ID})
	require.NoError(t, err)
	require.NoError(t, svc.ProcessNotificationRetryFromQueue(message.NewMessage("x", body)))
	assert.Empty(t, repo.lastUpdateID)
	assert.Empty(t, repo.recordFailureIDs)
}
