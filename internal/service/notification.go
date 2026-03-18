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
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
	"github.com/hiamthach108/dreon-notification/pkg/validator"
)

type INotificationSvc interface {
	CreateNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.NotificationAggregate, error)
	SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error)
	EnqueueNotification(ctx context.Context, req *aggregate.SendNotificationReq) (string, error)
	ProcessNotificationFromQueue(msg *message.Message) error
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
	}
}

func (s *NotificationSvc) CreateNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.NotificationAggregate, error) {
	notification := s.buildNotificationModel(req)
	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("marshal params: %w", err))
	}
	notification.Params = paramsJSON

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
	notification := s.buildNotificationModel(req)
	paramsJSON, _ := json.Marshal(req.Params)
	notification.Params = paramsJSON

	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		return "", err
	}
	if created == nil || created.ID == "" {
		return "", errorx.New(errorx.ErrInternal, "notification created but ID is empty")
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
	_, err := s.SendNotification(ctx, &payload.Req)
	if err != nil {
		_ = s.repo.Update(ctx, payload.NotificationID, model.Notification{Status: model.NotificationStatusFailed}, "Status")
		s.logger.Error("notification send failed, message committed for later handling", "notification_id", payload.NotificationID, "message_uuid", msg.UUID, "error", err)
		return nil
	}
	now := time.Now()
	if err := s.repo.Update(ctx, payload.NotificationID, model.Notification{
		Status: model.NotificationStatusCompleted,
		SentAt: now,
	}, "Status", "SentAt"); err != nil {
		s.logger.Error("failed to update notification status, message committed", "notification_id", payload.NotificationID, "message_uuid", msg.UUID, "error", err)
		return nil
	}
	s.logger.Info("notification processed and message committed", "notification_id", payload.NotificationID, "message_uuid", msg.UUID)
	return nil
}

func (s *NotificationSvc) buildNotificationModel(req *aggregate.SendNotificationReq) *model.Notification {
	n := &model.Notification{
		IdempotencyKey: req.IdempotencyKey,
		Source:         req.Source,
		Channel:        model.NotificationChannel(req.Channel),
		Type:           model.NotificationType(req.Type),
		Status:         model.NotificationStatusPending,
		Title:          req.Title,
		Message:        req.Message,
		Recipients:     req.Recipients,
		Provider:       channelToProvider(req.Channel),
	}
	if req.ScheduledAt != nil {
		n.ScheduledAt = *req.ScheduledAt
	}
	if req.ExpiredAt != nil {
		n.ExpiredAt = *req.ExpiredAt
	}
	return n
}

func channelToProvider(channel string) model.NotificationProvider {
	switch channel {
	case string(model.NotificationChannelEmail):
		return model.NotificationProviderResend
	case string(model.NotificationChannelSms):
		return model.NotificationProviderTwilio
	case string(model.NotificationChannelPush), string(model.NotificationChannelInApp):
		return model.NotificationProviderFirebase
	default:
		return model.NotificationProviderResend
	}
}

func (s *NotificationSvc) SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
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
		return nil, errorx.New(errorx.ErrInternal, fmt.Sprintf("unsupported channel: %s", req.Channel))
	}
}

func (s *NotificationSvc) sendEmail(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	templateName, ok := constant.EmailTemplateMap[req.Type]
	if !ok {
		return nil, errorx.New(errorx.ErrInternal, fmt.Sprintf("no email template for type: %s", req.Type))
	}
	params := req.Params
	if params == nil {
		params = make(map[string]any)
	}
	html, err := s.renderer.Render(ctx, templateName, params)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("render email template: %w", err))
	}
	data := &email.EmailData{
		From:    s.fromEmail,
		To:      req.Recipients,
		Subject: req.Title,
		HTML:    html,
	}
	err = s.emailClient.SendEmail(ctx, data)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("send email: %w", err))
	}
	// TODO: persist notification record via repo and return NotificationID
	return &aggregate.SendNotificationResp{NotificationID: ""}, nil
}

func (s *NotificationSvc) sendSMS(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	if s.smsClient == nil {
		return nil, errorx.New(errorx.ErrInternal, "SMS client not configured")
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
			return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("render sms template: %w", err))
		}
	} else {
		body = req.Message
	}
	if body == "" {
		return nil, errorx.New(errorx.ErrInternal, "SMS body is empty")
	}
	data := &sms.SMSData{
		To:   req.Recipients,
		Body: body,
	}
	if err := s.smsClient.SendSMS(ctx, data); err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("send sms: %w", err))
	}
	return &aggregate.SendNotificationResp{NotificationID: ""}, nil
}

func (s *NotificationSvc) sendPush(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	return nil, errorx.New(errorx.ErrInternal, "PUSH channel not implemented")
}

func (s *NotificationSvc) sendInApp(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	return nil, errorx.New(errorx.ErrInternal, "IN_APP channel not implemented")
}
