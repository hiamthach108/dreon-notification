package aggregate

import "time"

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
