package aggregate

import (
	"encoding/json"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
)

// RegisterUserFCMTokenReq registers or updates a device token for a user (CreatedBy = UserID).
type RegisterUserFCMTokenReq struct {
	UserID         string         `json:"userId" validate:"required,uuid"`
	Token          string         `json:"token" validate:"required,min=10,max=4096"`
	Platform       string         `json:"platform" validate:"required,oneof=IOS ANDROID WEB"`
	DeviceMetadata map[string]any `json:"deviceMetadata,omitempty"`
}

// ListUserFCMTokenQuery binds GET /fcm-tokens?userId=...
type ListUserFCMTokenQuery struct {
	UserID string `query:"userId" validate:"required,uuid"`
}

// DeleteUserFCMTokenQuery binds DELETE /fcm-tokens/:id?userId=...
type DeleteUserFCMTokenQuery struct {
	UserID string `query:"userId" validate:"required,uuid"`
}

// UserFCMTokenAggregate is the API-facing shape of a stored FCM token row.
type UserFCMTokenAggregate struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId"`
	Token          string         `json:"token"`
	Platform       string         `json:"platform"`
	DeviceMetadata map[string]any `json:"deviceMetadata,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (a *UserFCMTokenAggregate) FromModel(m *model.UserFCMToken) {
	if m == nil || a == nil {
		return
	}
	a.ID = m.ID
	a.UserID = m.CreatedBy
	a.Token = m.Token
	a.Platform = m.Platform
	if len(m.DeviceMetadata) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(m.DeviceMetadata, &meta); err == nil && meta != nil {
			a.DeviceMetadata = meta
		}
	}
	a.CreatedAt = m.CreatedAt
	a.UpdatedAt = m.UpdatedAt
}
