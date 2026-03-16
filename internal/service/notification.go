package service

import (
	"context"
	"fmt"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/email"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
)

// emailTemplateMap maps notification type to MJML template name (without .mjml) for EMAIL channel.
var emailTemplateMap = map[string]string{
	string(model.NotificationTypeWelcome):        "welcome",
	string(model.NotificationTypeVerifyOTP):      "verify-otp",
	string(model.NotificationTypeForgotPassword): "forgot-password",
	string(model.NotificationTypeResetPassword):  "reset-password",
}

type INotificationSvc interface {
	SendNotification(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error)
}

type NotificationSvc struct {
	logger      logger.ILogger
	repo        repository.INotificationRepository
	emailClient email.IEmailClient
	renderer    email.IRenderer
	fromEmail   string
}

// NewNotificationSvc builds the notification service. Sender for EMAIL channel is read from cfg.Email.Sender.
func NewNotificationSvc(
	logger logger.ILogger,
	repo repository.INotificationRepository,
	emailClient email.IEmailClient,
	renderer email.IRenderer,
	cfg *config.AppConfig,
) INotificationSvc {
	return &NotificationSvc{
		logger:      logger,
		repo:        repo,
		emailClient: emailClient,
		renderer:    renderer,
		fromEmail:   cfg.Email.Sender,
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
	templateName, ok := emailTemplateMap[req.Type]
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
	return nil, errorx.New(errorx.ErrInternal, "SMS channel not implemented")
}

func (s *NotificationSvc) sendPush(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	return nil, errorx.New(errorx.ErrInternal, "PUSH channel not implemented")
}

func (s *NotificationSvc) sendInApp(ctx context.Context, req *aggregate.SendNotificationReq) (*aggregate.SendNotificationResp, error) {
	return nil, errorx.New(errorx.ErrInternal, "IN_APP channel not implemented")
}
