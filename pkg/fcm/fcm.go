package fcm

import (
	"context"
	"errors"
)

// Common client errors.
var (
	ErrMissingCredentials = errors.New("missing Firebase credentials path")
	ErrNoTokens           = errors.New("no FCM registration tokens")
	ErrEmptyMessage       = errors.New("message must include title/body and/or data payload")
)

// PushMessage is the notification payload sent to each token.
type PushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// SendOutcome aggregates results across multicast batches (FCM allows up to 500 tokens per call).
type SendOutcome struct {
	SuccessCount int
	FailureCount int
}

// IFCMClient sends FCM messages to device registration tokens.
type IFCMClient interface {
	SendToTokens(ctx context.Context, tokens []string, msg *PushMessage) (*SendOutcome, error)
}
