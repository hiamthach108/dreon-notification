package email

import "context"

type EmailData struct {
	From    string
	To      []string
	Subject string
	HTML    string
}

type IEmailClient interface {
	SendEmail(ctx context.Context, email *EmailData) error
}
