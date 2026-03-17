package sms

import "context"

// SMSData holds data required to send an SMS (one or more recipients, same body).
type SMSData struct {
	To   []string // E.164 phone numbers, e.g. +15558675309
	Body string
}

// ISMSClient sends SMS via a provider (e.g. Twilio).
type ISMSClient interface {
	SendSMS(ctx context.Context, data *SMSData) error
}
