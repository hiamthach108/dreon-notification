package aggregate

import (
	"encoding/json"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
)

type SendNotificationReq struct {
	IdempotencyKey string         `json:"idempotencyKey" validate:"required,min=1,max=255"`
	Source         string         `json:"source" validate:"required,min=1,max=255"`
	Channel        string         `json:"channel" validate:"oneof=EMAIL SMS PUSH IN_APP"`
	Type           string         `json:"type" validate:"oneof=WELCOME VERIFY_OTP FORGOT_PASSWORD RESET_PASSWORD"`
	Title          string         `json:"title" validate:"required,min=1,max=255"`
	Message        string         `json:"message"`
	Recipients     []string       `json:"recipients" validate:"required,min=1,max=50"`
	Params         map[string]any `json:"params"`
	ScheduledAt    *time.Time     `json:"scheduledAt" validate:"omitempty,datetime"`
	ExpiredAt      *time.Time     `json:"expiredAt" validate:"omitempty,datetime"`
}

type SendNotificationResp struct {
	NotificationID string `json:"notificationId"`
}

// NotificationEnqueuePayload is the message payload published to the notifications queue.
// Used by the service when publishing and by the consumer when unmarshalling.
type NotificationEnqueuePayload struct {
	NotificationID string             `json:"notificationId"`
	Req            SendNotificationReq `json:"req"`
}

type NotificationAggregate struct {
	ID             string         `json:"id"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Source         string         `json:"source"`
	Channel        string         `json:"channel"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	Recipients     []string       `json:"recipients"`
	Params         map[string]any `json:"params"`
	Provider       string         `json:"provider"`
	ProviderID     string         `json:"providerId"`
	ExpiredAt      *time.Time     `json:"expiredAt"`
	SentAt         *time.Time     `json:"sentAt"`
	ScheduledAt    *time.Time     `json:"scheduledAt"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      *time.Time     `json:"deletedAt"`
	CreatedBy      string         `json:"createdBy"`
	UpdatedBy      string         `json:"updatedBy"`
}

func (n *NotificationAggregate) FromModel(model *model.Notification) {
	n.ID = model.ID
	n.IdempotencyKey = model.IdempotencyKey
	n.Source = model.Source
	n.Channel = string(model.Channel)
	n.Type = string(model.Type)
	n.Status = string(model.Status)
	n.Title = model.Title
	n.Message = model.Message
	n.Recipients = model.Recipients
	n.Provider = string(model.Provider)
	n.ProviderID = model.ProviderID
	if model.Params != nil {
		var params map[string]any
		if err := json.Unmarshal(model.Params, &params); err != nil {
			n.Params = make(map[string]any)
		} else {
			n.Params = params
		}
	}

	if model.ExpiredAt != (time.Time{}) {
		n.ExpiredAt = &model.ExpiredAt
	}
	if model.SentAt != (time.Time{}) {
		n.SentAt = &model.SentAt
	}
	if model.ScheduledAt != (time.Time{}) {
		n.ScheduledAt = &model.ScheduledAt
	}
	n.CreatedAt = model.CreatedAt
	n.UpdatedAt = model.UpdatedAt
	n.DeletedAt = &model.DeletedAt.Time
	n.CreatedBy = model.CreatedBy
	n.UpdatedBy = model.UpdatedBy
}
