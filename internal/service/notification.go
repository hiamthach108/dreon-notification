package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/internal/shared/constant"
	"github.com/hiamthach108/dreon-notification/pkg/cache"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/fcm"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
	"github.com/hiamthach108/dreon-notification/pkg/validator"
	"gorm.io/gorm"
)

type INotificationSvc interface {
	CreateNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.NotificationAggregate, error)
	SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) error
	EnqueueNotification(ctx context.Context, req *aggregate.SendNotificationReq) (string, error)
	ProcessNotificationFromQueue(msg *message.Message) error
	ProcessNotificationRetryFromQueue(msg *message.Message) error
	EnqueuePendingRetries(ctx context.Context, batchSize int) error
	EnqueueDueScheduledNotifications(ctx context.Context, batchSize int) error
}

type NotificationSvc struct {
	logger        logger.ILogger
	repo          repository.INotificationRepository
	publisher     *amqp.Publisher
	emailClient   email.IEmailClient
	renderer      email.IRenderer
	fromEmail     string
	smsClient     sms.ISMSClient
	smsBodyRender sms.IBodyRenderer
	fcmClient     fcm.IFCMClient
	cache         cache.ICache
	cfg           *config.AppConfig
}

// NewNotificationSvc builds the notification service. Sender for EMAIL channel is read from cfg.Email.Sender.
func NewNotificationSvc(
	logger logger.ILogger,
	repo repository.INotificationRepository,
	publisher *amqp.Publisher,
	emailClient email.IEmailClient,
	renderer email.IRenderer,
	smsClient sms.ISMSClient,
	smsBodyRender sms.IBodyRenderer,
	fcmClient fcm.IFCMClient,
	appCache cache.ICache,
	cfg *config.AppConfig,
) INotificationSvc {
	return &NotificationSvc{
		logger:        logger,
		repo:          repo,
		publisher:     publisher,
		emailClient:   emailClient,
		renderer:      renderer,
		fromEmail:     cfg.Email.Sender,
		smsClient:     smsClient,
		smsBodyRender: smsBodyRender,
		fcmClient:     fcmClient,
		cache:         appCache,
		cfg:           cfg,
	}
}

func (s *NotificationSvc) CreateNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.NotificationAggregate, error) {
	notification := req.ToModel()
	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("marshal params: %w", err))
	}
	notification.Params = paramsJSON
	notification.MaxAttempts = s.cfg.Notification.MaxAttempts

	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		return nil, err
	}
	if created == nil || created.ID == "" {
		return nil, errorx.New(errorx.ErrInternal, "notification created but ID is empty")
	}

	agg := &aggregate.NotificationAggregate{}
	agg.FromModel(created)
	return agg, nil
}

func (s *NotificationSvc) EnqueueNotification(ctx context.Context, req *aggregate.SendNotificationReq) (string, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return "", errorx.Wrap(errorx.ErrBadRequest, validator.FormatValidationError(err))
	}
	notification := req.ToModel()
	paramsJSON, _ := json.Marshal(req.Params)
	notification.Params = paramsJSON
	notification.MaxAttempts = s.cfg.Notification.MaxAttempts

	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		return "", err
	}
	if created == nil || created.ID == "" {
		return "", errorx.New(errorx.ErrInternal, "notification created but ID is empty")
	}
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now().UTC()) {
		return created.ID, nil
	}
	if s.publisher == nil {
		return "", errorx.New(errorx.ErrInternal, "message publisher not configured")
	}

	payload := aggregate.NotificationEnqueuePayload{
		NotificationID: created.ID,
		Req:            *req,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", errorx.Wrap(errorx.ErrInternal, fmt.Errorf("marshal enqueue payload: %w", err))
	}
	msg := message.NewMessage(created.ID, payloadBytes)
	if err := s.publisher.Publish(constant.EventTopicNotificationsSend, msg); err != nil {
		return "", errorx.Wrap(errorx.ErrInternal, fmt.Errorf("publish to queue: %w", err))
	}
	return created.ID, nil
}

func (s *NotificationSvc) ProcessNotificationFromQueue(msg *message.Message) error {
	ctx := context.Background()
	var payload aggregate.NotificationEnqueuePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		s.logger.Error("invalid queue payload, message committed", "message_uuid", msg.UUID, "error", err)
		return nil
	}
	err := s.SendNotification(ctx, &payload.Req)
	if err != nil {
		initial, max := s.backoffParams()
		if recErr := s.repo.RecordSendFailure(ctx, payload.NotificationID, initial, max); recErr != nil {
			s.logger.Error("record send failure after queue send error", "notification_id", payload.NotificationID, "message_uuid", msg.UUID, "error", recErr)
		}
		s.logger.Error("notification send failed, message committed for later handling", "notification_id", payload.NotificationID, "message_uuid", msg.UUID, "error", err)
		return nil
	}
	now := time.Now()
	if err := s.repo.Update(ctx, payload.NotificationID, model.Notification{
		Status:       model.NotificationStatusCompleted,
		SentAt:       now,
		AttemptCount: 1,
	}, "Status", "SentAt", "AttemptCount"); err != nil {
		s.logger.Error("failed to update notification status, message committed", "notification_id", payload.NotificationID, "message_uuid", msg.UUID, "error", err)
		return nil
	}
	s.logger.Info("notification processed and message committed", "notification_id", payload.NotificationID, "message_uuid", msg.UUID)
	return nil
}

func (s *NotificationSvc) EnqueueDueScheduledNotifications(ctx context.Context, batchSize int) error {
	if batchSize <= 0 {
		return nil
	}
	if s.publisher == nil {
		return errorx.New(errorx.ErrInternal, "message publisher not configured")
	}
	if s.cache == nil {
		return errorx.New(errorx.ErrInternal, "cache not configured")
	}
	now := time.Now().UTC()
	rows, err := s.repo.FindDueScheduledNotifications(ctx, batchSize, now)
	if err != nil {
		return fmt.Errorf("find due scheduled notifications: %w", err)
	}
	ttlSec := s.cfg.Notification.ScheduledDedupTTLSec
	if ttlSec <= 0 {
		ttlSec = 86400
	}
	ttl := time.Duration(ttlSec) * time.Second
	for i := range rows {
		n := rows[i]
		dedupKey := constant.CacheKeyScheduledEnqueueDedup + n.ID
		ok, nxErr := s.cache.SetNX(ctx, dedupKey, ttl)
		if nxErr != nil {
			s.logger.Error("scheduled enqueue dedup setnx failed", "notification_id", n.ID, "error", nxErr)
			continue
		}
		if !ok {
			continue
		}
		req := &aggregate.SendNotificationReq{}
		req.FromModel(&n)
		payload := aggregate.NotificationEnqueuePayload{
			NotificationID: n.ID,
			Req:            *req,
		}
		payloadBytes, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			s.logger.Error("marshal scheduled enqueue payload", "notification_id", n.ID, "error", marshalErr)
			continue
		}
		msg := message.NewMessage(n.ID, payloadBytes)
		if pubErr := s.publisher.Publish(constant.EventTopicNotificationsSend, msg); pubErr != nil {
			s.logger.Error("publish scheduled notification to send topic failed", "notification_id", n.ID, "error", pubErr)
			continue
		}
		s.logger.Info("enqueued scheduled notification send", "notification_id", n.ID)
	}
	return nil
}

func (s *NotificationSvc) EnqueuePendingRetries(ctx context.Context, batchSize int) error {
	if batchSize <= 0 {
		return nil
	}
	if s.publisher == nil {
		return errorx.New(errorx.ErrInternal, "message publisher not configured")
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(s.publishLease())
	var rows []model.Notification
	if err := s.repo.RunInTransaction(ctx, func(tx *gorm.DB) error {
		found, err := s.repo.LockPendingRetriesDueForUpdate(tx, batchSize, now)
		if err != nil {
			return err
		}
		for i := range found {
			if err := s.repo.UpdateNextRetryAt(tx, found[i].ID, leaseUntil); err != nil {
				return err
			}
			found[i].NextRetryAt = &leaseUntil
		}
		rows = found
		return nil
	}); err != nil {
		return fmt.Errorf("claim pending retries for publish: %w", err)
	}
	for i := range rows {
		n := rows[i]
		payload := aggregate.NotificationRetryPayload{NotificationID: n.ID}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			s.logger.Error("marshal retry payload", "notification_id", n.ID, "error", err)
			continue
		}
		msg := message.NewMessage(n.ID, payloadBytes)
		if err := s.publisher.Publish(constant.EventTopicNotificationsRetry, msg); err != nil {
			s.logger.Error("publish to retry topic failed", "notification_id", n.ID, "error", err)
			continue
		}
		s.logger.Info("enqueued notification retry", "notification_id", n.ID)
	}
	return nil
}

func (s *NotificationSvc) ProcessNotificationRetryFromQueue(msg *message.Message) error {
	ctx := context.Background()
	var payload aggregate.NotificationRetryPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		s.logger.Error("invalid retry queue payload, message committed", "message_uuid", msg.UUID, "error", err)
		return nil
	}
	if payload.NotificationID == "" {
		s.logger.Error("retry queue payload missing notification id", "message_uuid", msg.UUID)
		return nil
	}
	n := s.repo.FindOneById(ctx, payload.NotificationID)
	if n == nil {
		s.logger.Error("retry notification row not found", "notification_id", payload.NotificationID, "message_uuid", msg.UUID)
		return nil
	}
	if n.Status != model.NotificationStatusPending {
		return nil
	}
	if n.AttemptCount < 1 || n.AttemptCount >= n.MaxAttempts {
		return nil
	}
	req := &aggregate.SendNotificationReq{}
	req.FromModel(n)
	initial, max := s.backoffParams()
	sendErr := s.SendNotification(ctx, req)
	if sendErr != nil {
		if recErr := s.repo.RecordSendFailure(ctx, n.ID, initial, max); recErr != nil {
			s.logger.Error("record send failure after retry queue send error", "notification_id", n.ID, "message_uuid", msg.UUID, "error", recErr)
		}
		s.logger.Error("notification retry send failed, message committed", "notification_id", n.ID, "message_uuid", msg.UUID, "error", sendErr)
		return nil
	}
	now := time.Now()
	if err := s.repo.Update(ctx, n.ID, model.Notification{
		Status: model.NotificationStatusCompleted,
		SentAt: now,
	}, "Status", "SentAt"); err != nil {
		s.logger.Error("failed to update notification after retry success", "notification_id", n.ID, "message_uuid", msg.UUID, "error", err)
		return nil
	}
	s.logger.Info("notification retry processed and message committed", "notification_id", n.ID, "message_uuid", msg.UUID)
	return nil
}

func (s *NotificationSvc) backoffParams() (initialSec, maxSec int) {
	initialSec = s.cfg.Notification.RetryBackoffInitialSec
	if initialSec <= 0 {
		initialSec = 30
	}
	maxSec = s.cfg.Notification.RetryBackoffMaxSec
	if maxSec <= 0 {
		maxSec = 3600
	}
	if maxSec < initialSec {
		maxSec = initialSec
	}
	return initialSec, maxSec
}

func (s *NotificationSvc) publishLease() time.Duration {
	sec := s.cfg.Notification.RetryPublishLeaseSec
	if sec <= 0 {
		sec = 300
	}
	return time.Duration(sec) * time.Second
}

func (s *NotificationSvc) SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) error {
	switch req.Channel {
	case string(model.NotificationChannelEmail):
		return s.sendEmail(ctx, req)
	case string(model.NotificationChannelSms):
		return s.sendSMS(ctx, req)
	case string(model.NotificationChannelPush):
		return s.sendPush(ctx, req)
	case string(model.NotificationChannelInApp):
		return s.sendInApp(ctx, req)
	default:
		return errorx.New(errorx.ErrInternal, fmt.Sprintf("unsupported channel: %s", req.Channel))
	}
}

func (s *NotificationSvc) sendEmail(ctx context.Context, req *aggregate.SendNotificationReq) error {
	templateName, ok := constant.EmailTemplateMap[req.Type]
	if !ok {
		return errorx.New(errorx.ErrInternal, fmt.Sprintf("no email template for type: %s", req.Type))
	}
	params := req.Params
	if params == nil {
		params = make(map[string]any)
	}
	html, err := s.renderer.Render(ctx, templateName, params)
	if err != nil {
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("render email template: %w", err))
	}
	data := &email.EmailData{
		From:    s.fromEmail,
		To:      req.Recipients,
		Subject: req.Title,
		HTML:    html,
	}
	err = s.emailClient.SendEmail(ctx, data)
	if err != nil {
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("send email: %w", err))
	}
	return nil
}

func (s *NotificationSvc) sendSMS(ctx context.Context, req *aggregate.SendNotificationReq) error {
	if s.smsClient == nil {
		return errorx.New(errorx.ErrInternal, "SMS client not configured")
	}
	params := req.Params
	if params == nil {
		params = make(map[string]any)
	}
	var body string
	if templateName, ok := constant.SMSTemplateMap[req.Type]; ok && s.smsBodyRender != nil {
		var err error
		body, err = s.smsBodyRender.RenderBody(ctx, templateName, params)
		if err != nil {
			return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("render sms template: %w", err))
		}
	} else {
		body = req.Message
	}
	if body == "" {
		return errorx.New(errorx.ErrInternal, "SMS body is empty")
	}
	data := &sms.SMSData{
		To:   req.Recipients,
		Body: body,
	}
	if err := s.smsClient.SendSMS(ctx, data); err != nil {
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("send sms: %w", err))
	}
	return nil
}

func (s *NotificationSvc) sendPush(ctx context.Context, req *aggregate.SendNotificationReq) error {
	if s.fcmClient == nil {
		return errorx.New(errorx.ErrInternal, "FCM client not configured")
	}
	msg := &fcm.PushMessage{
		Title: req.Title,
		Body:  req.Message,
	}
	outcome, err := s.fcmClient.SendToTokens(ctx, req.Recipients, msg)
	if err != nil {
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("send push: %w", err))
	}
	if outcome.SuccessCount == 0 {
		return errorx.New(errorx.ErrInternal, "no FCM tokens were successfully sent")
	}
	if outcome.FailureCount > 0 {
		s.logger.Error("failed to send push to some tokens", "success_count", outcome.SuccessCount, "failure_count", outcome.FailureCount)
	}
	s.logger.Info("push sent", "success_count", outcome.SuccessCount, "failure_count", outcome.FailureCount)
	return nil
}

func (s *NotificationSvc) sendInApp(ctx context.Context, req *aggregate.SendNotificationReq) error {
	return errorx.New(errorx.ErrInternal, "IN_APP channel not implemented")
}
