package service

import (
	"context"
	"fmt"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/internal/shared/constant"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/sms"
)

type INotificationSvc interface {
	SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error)
}

type NotificationSvc struct {
	logger        logger.ILogger
	repo          repository.INotificationRepository
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
	emailClient email.IEmailClient,
	renderer email.IRenderer,
	smsClient sms.ISMSClient,
	smsBodyRender sms.IBodyRenderer,
	cfg *config.AppConfig,
) INotificationSvc {
	return &NotificationSvc{
		logger:        logger,
		repo:          repo,
		emailClient:   emailClient,
		renderer:      renderer,
		fromEmail:     cfg.Email.Sender,
		smsClient:     smsClient,
		smsBodyRender: smsBodyRender,
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
