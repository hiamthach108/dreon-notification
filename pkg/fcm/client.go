package fcm

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/pkg/logger"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// maxTokensPerMulticast is the FCM limit for a single multicast request.
const maxTokensPerMulticast = 500

type firebaseClient struct {
	messaging *messaging.Client
	logger    logger.ILogger
}

// NewClient builds an FCM client using a Firebase service account JSON file path.
func NewClient(cfg *config.AppConfig, log logger.ILogger) (IFCMClient, error) {
	ctx := context.Background()
	if strings.TrimSpace(cfg.Firebase.CredentialsPath) == "" {
		return nil, fmt.Errorf("fcm: %w", ErrMissingCredentials)
	}
	if log == nil {
		return nil, fmt.Errorf("fcm: logger is required")
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cfg.Firebase.CredentialsPath))
	if err != nil {
		return nil, fmt.Errorf("fcm: init firebase app: %w", err)
	}

	msg, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: init messaging: %w", err)
	}

	return &firebaseClient{messaging: msg, logger: log}, nil
}

func (c *firebaseClient) SendToTokens(ctx context.Context, tokens []string, msg *PushMessage) (*SendOutcome, error) {
	filtered := filterNonEmptyTokens(tokens)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("fcm: %w", ErrNoTokens)
	}
	if err := validatePushMessage(msg); err != nil {
		return nil, err
	}

	var outcome SendOutcome
	for i := 0; i < len(filtered); i += maxTokensPerMulticast {
		end := min(i+maxTokensPerMulticast, len(filtered))
		batch := filtered[i:end]

		mm := toMulticastMessage(batch, msg)

		br, err := c.messaging.SendEachForMulticast(ctx, mm)
		if err != nil {
			return &outcome, fmt.Errorf("fcm: send multicast: %w", err)
		}

		outcome.SuccessCount += br.SuccessCount
		outcome.FailureCount += br.FailureCount

		if br.FailureCount > 0 {
			c.logger.Warn("FCM multicast partial failure",
				"success", br.SuccessCount,
				"failure", br.FailureCount,
				"batchSize", len(batch),
			)
		} else {
			c.logger.Info("FCM multicast sent",
				"success", br.SuccessCount,
				"batchSize", len(batch),
			)
		}
	}

	return &outcome, nil
}

func (c *firebaseClient) SendToTopics(ctx context.Context, topics []string, msg *PushMessage) (*SendOutcome, error) {
	filtered := filterNonEmptyTopics(topics)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("fcm: %w", ErrNoTopics)
	}
	if err := validatePushMessage(msg); err != nil {
		return nil, err
	}

	var outcome SendOutcome
	for _, topic := range filtered {
		tm := toTopicMessage(topic, msg)
		_, err := c.messaging.Send(ctx, tm)
		if err != nil {
			outcome.FailureCount++
			c.logger.Warn("FCM topic send failed", "topic", topic, "error", err)
			continue
		}
		outcome.SuccessCount++
		c.logger.Info("FCM topic sent", "topic", topic)
	}

	if outcome.SuccessCount == 0 {
		return &outcome, fmt.Errorf("fcm: no topics were successfully sent")
	}
	if outcome.FailureCount > 0 {
		c.logger.Warn("FCM topic partial failure",
			"success", outcome.SuccessCount,
			"failure", outcome.FailureCount,
		)
	}
	return &outcome, nil
}

func validatePushMessage(msg *PushMessage) error {
	if msg == nil {
		return fmt.Errorf("fcm: message is nil")
	}
	hasNotif := msg.Title != "" || msg.Body != ""
	if !hasNotif && len(msg.Data) == 0 {
		return fmt.Errorf("fcm: %w", ErrEmptyMessage)
	}
	return nil
}

func toMulticastMessage(tokens []string, msg *PushMessage) *messaging.MulticastMessage {
	mm := &messaging.MulticastMessage{Tokens: tokens}
	if len(msg.Data) > 0 {
		mm.Data = maps.Clone(msg.Data)
	}
	if msg.Title != "" || msg.Body != "" {
		mm.Notification = &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		}
	}
	return mm
}

func toTopicMessage(topic string, msg *PushMessage) *messaging.Message {
	m := &messaging.Message{Topic: topic}
	if len(msg.Data) > 0 {
		m.Data = maps.Clone(msg.Data)
	}
	if msg.Title != "" || msg.Body != "" {
		m.Notification = &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		}
	}
	return m
}

func filterNonEmptyTopics(topics []string) []string {
	out := make([]string, 0, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func filterNonEmptyTokens(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
