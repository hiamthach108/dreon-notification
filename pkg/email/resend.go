package email

import (
	"context"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/pkg/logger"

	"github.com/resend/resend-go/v3"
)

type ResendEmailClient struct {
	logger logger.ILogger
	config *config.AppConfig

	resendClient *resend.Client
}

func NewResendEmailClient(config *config.AppConfig, logger logger.ILogger) IEmailClient {
	return &ResendEmailClient{
		logger:       logger,
		config:       config,
		resendClient: resend.NewClient(config.Email.ResendAPIKey),
	}
}

func (c *ResendEmailClient) SendEmail(ctx context.Context, email *EmailData) error {
	c.logger.Info("Sending email", "from", c.config.Email.Sender, "to", email.To)

	params := &resend.SendEmailRequest{
		From:    c.config.Email.Sender,
		To:      email.To,
		Subject: email.Subject,
		Html:    email.HTML,
	}

	resp, err := c.resendClient.Emails.Send(params)
	if err != nil {
		c.logger.Error("Failed to send email", "error", err)
		return err
	}

	c.logger.Info("Email sent successfully", "response", resp.Id)

	return nil
}
