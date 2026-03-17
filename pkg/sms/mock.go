package sms

import (
	"context"
	"fmt"
)

// MockClient is an ISMSClient that returns an error on SendSMS. Use when Twilio is not configured or in tests.
type MockClient struct{}

// NewMockClient returns a client that returns an error on SendSMS (SMS not configured).
func NewMockClient() ISMSClient {
	return &MockClient{}
}

// SendSMS implements ISMSClient.
func (MockClient) SendSMS(ctx context.Context, data *SMSData) error {
	return fmt.Errorf("sms: client not configured (set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER)")
}

var _ ISMSClient = (*MockClient)(nil)
