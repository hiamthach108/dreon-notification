package model

import (
	"time"

	"gorm.io/datatypes"
)

type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
	NotificationChannelSms   NotificationChannel = "SMS"
	NotificationChannelPush  NotificationChannel = "PUSH"
	NotificationChannelInApp NotificationChannel = "IN_APP"
)

type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "PENDING"
	NotificationStatusCompleted NotificationStatus = "COMPLETED"
	NotificationStatusFailed    NotificationStatus = "FAILED"
)

type NotificationProvider string

const (
	NotificationProviderFirebase NotificationProvider = "FIREBASE"
	NotificationProviderAwsSes   NotificationProvider = "AWS_SES"
	NotificationProviderTwilio   NotificationProvider = "TWILIO"
	NotificationProviderSmtp     NotificationProvider = "SMTP"
)

type NotificationType string

const (
	NotificationTypeWelcome        NotificationType = "WELCOME"
	NotificationTypeVerifyOTP      NotificationType = "VERIFY_OTP"
	NotificationTypeForgotPassword NotificationType = "FORGOT_PASSWORD"
	NotificationTypeResetPassword  NotificationType = "RESET_PASSWORD"
)

type Notification struct {
	BaseModel

	IdempotencyKey string              `gorm:"type:varchar(255);not null;uniqueIndex:idx_notification_idempotency_key"`
	Source         string              `gorm:"type:varchar(255);not null"`
	Channel        NotificationChannel `gorm:"type:varchar(255);not null;index:idx_notification_channel_type,priority:1"`
	Type           NotificationType    `gorm:"type:varchar(255);not null;index:idx_notification_channel_type,priority:2"`
	Status         NotificationStatus  `gorm:"type:varchar(255);not null"`

	Title      string                      `gorm:"type:varchar(255);not null"`
	Message    string                      `gorm:"type:text;"`
	Recipients datatypes.JSONSlice[string] `gorm:"type:jsonb;"`
	Params     datatypes.JSON              `gorm:"type:jsonb;"`

	Provider   NotificationProvider `gorm:"type:varchar(255);not null"`
	ProviderID string               `gorm:"type:varchar(255)"`

	ExpiredAt   time.Time `gorm:"type:timestamp;"`
	SentAt      time.Time `gorm:"type:timestamp;"`
	ScheduledAt time.Time `gorm:"type:timestamp;"`
	
}
