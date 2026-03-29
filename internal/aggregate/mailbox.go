package aggregate

import (
	"strings"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/model"
)

// CreateMailboxReq is the input for creating an in-app mailbox row for a user.
// UserID is the owning user and is persisted as model.BaseModel.CreatedBy.
type CreateMailboxReq struct {
	UserID         string `validate:"required,uuid"`
	Title          string `validate:"required,max=255"`
	Message        string `validate:"omitempty"`
	Group          string `validate:"omitempty,max=255"`
	NotificationID string `validate:"required,uuid"`
}

func (req *CreateMailboxReq) ToModel() *model.Mailbox {
	m := &model.Mailbox{
		Title:          strings.TrimSpace(req.Title),
		Message:        req.Message,
		Group:          strings.TrimSpace(req.Group),
		NotificationID: req.NotificationID,
	}
	m.CreatedBy = req.UserID
	return m
}

// MailboxAggregate is the API-facing view of a user's mailbox item.
type MailboxAggregate struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	IsRead         bool       `json:"isRead"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
	Group          string     `json:"group,omitempty"`
	NotificationID string     `json:"notificationId"`
	CreatedBy      string     `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (a *MailboxAggregate) FromModel(m *model.Mailbox) {
	if m == nil || a == nil {
		return
	}
	a.ID = m.ID
	a.Title = m.Title
	a.Message = m.Message
	a.IsRead = m.IsRead
	a.ReadAt = m.ReadAt
	a.Group = m.Group
	a.NotificationID = m.NotificationID
	a.CreatedBy = m.CreatedBy
	a.CreatedAt = m.CreatedAt
	a.UpdatedAt = m.UpdatedAt
}
