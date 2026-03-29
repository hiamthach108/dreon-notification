package model

import "gorm.io/datatypes"

// UserFCMToken stores an FCM registration token for a user device.
// BaseModel.CreatedBy holds the user ID (owner).
type UserFCMToken struct {
	BaseModel

	Token          string         `gorm:"type:text;not null;uniqueIndex:idx_user_fcm_tokens_token"`
	Platform       string         `gorm:"type:varchar(32);not null"`
	DeviceMetadata datatypes.JSON `gorm:"type:jsonb"`
}

func (UserFCMToken) TableName() string {
	return "user_fcm_tokens"
}
