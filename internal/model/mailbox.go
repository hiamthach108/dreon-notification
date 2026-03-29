package model

import "time"

type Mailbox struct {
	BaseModel

	Title          string     `gorm:"type:varchar(255);not null"`
	Message        string     `gorm:"type:text;"`
	IsRead         bool       `gorm:"type:boolean;not null;default:false"`
	ReadAt         *time.Time `gorm:"type:timestamp;"`
	Group          string     `gorm:"type:varchar(255);"`
	NotificationID string     `gorm:"type:varchar(36);not null;index"`

	Notification Notification `gorm:"foreignKey:NotificationID;references:ID"`
}

func (Mailbox) TableName() string {
	return "mailboxes"
}
