package sms

import (
	"context"
	"fmt"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// TwilioClient sends SMS via Twilio. It implements ISMSClient.
type TwilioClient struct {
	logger logger.ILogger
	config *config.AppConfig
	client *twilio.RestClient
}

// NewTwilioClient creates an SMS client using Twilio with credentials from config.
func NewTwilioClient(cfg *config.AppConfig, logger logger.ILogger) (ISMSClient, error) {
	if cfg.SMS.TwilioAccountSID == "" || cfg.SMS.TwilioAuthToken == "" {
		return nil, fmt.Errorf("sms/twilio: TwilioAccountSID and TwilioAuthToken are required")
	}
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.SMS.TwilioAccountSID,
		Password: cfg.SMS.TwilioAuthToken,
	})
	return &TwilioClient{
		logger: logger,
		config: cfg,
		client: client,
	}, nil
}

// SendSMS implements ISMSClient. Sends the same body to each recipient.
func (c *TwilioClient) SendSMS(ctx context.Context, data *SMSData) error {
	if len(data.To) == 0 {
		return fmt.Errorf("sms/twilio: at least one recipient required")
	}
	from := c.config.SMS.TwilioFromNumber
	if from == "" {
		return fmt.Errorf("sms/twilio: TwilioFromNumber is required")
	}
	for _, to := range data.To {
		params := &openapi.CreateMessageParams{}
		params.SetTo(to)
		params.SetFrom(from)
		params.SetBody(data.Body)
		_, err := c.client.Api.CreateMessage(params)
		if err != nil {
			c.logger.Error("Failed to send SMS", "to", to, "error", err)
			return fmt.Errorf("send SMS to %s: %w", to, err)
		}
		c.logger.Info("SMS sent", "to", to)
	}
	return nil
}

var _ ISMSClient = (*TwilioClient)(nil)
