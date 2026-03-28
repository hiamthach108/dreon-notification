package aggregate

import (
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
)

// CreatePushTopicReq is the input for creating a push topic registry row (FCM topic name metadata).
type CreatePushTopicReq struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"omitempty,max=4000"`
	IsActive    *bool  `json:"isActive,omitempty"`
}

// UpdatePushTopicReq is a partial update; only non-nil fields are applied.
type UpdatePushTopicReq struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=4000"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

// PushTopicAggregate is the API-facing aggregate for a push topic.
type PushTopicAggregate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (a *PushTopicAggregate) FromModel(m *model.PushTopic) {
	if m == nil || a == nil {
		return
	}
	a.ID = m.ID
	a.Name = m.Name
	a.Description = m.Description
	a.IsActive = m.IsActive
	a.CreatedAt = m.CreatedAt
	a.UpdatedAt = m.UpdatedAt
}
