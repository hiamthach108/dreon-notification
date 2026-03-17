package constant

import (
	"github.com/hiamthach108/dreon-notification/internal/model"
)

// EmailTemplateMap maps notification type to MJML template name (without .mjml) for EMAIL channel.
var EmailTemplateMap = map[string]string{
	string(model.NotificationTypeWelcome):        "welcome",
	string(model.NotificationTypeVerifyOTP):     "verify-otp",
	string(model.NotificationTypeForgotPassword): "forgot-password",
	string(model.NotificationTypeResetPassword): "reset-password",
}

// SMSTemplateMap maps notification type to SMS body template name (without .txt) for SMS channel.
var SMSTemplateMap = map[string]string{
	string(model.NotificationTypeVerifyOTP): "verify-otp",
	// Add more as needed: welcome, forgot-password, etc.
}
