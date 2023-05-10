package login

import (
	"context"
	"mailboxsweep/email"
)

type EmailSender struct {
	client       email.Sender
	templateName string
}

func NewEmailSender(client email.Sender, templateName string) *EmailSender {
	return &EmailSender{client, templateName}
}

func (s EmailSender) Send(ctx context.Context, req SendRequest) error {
	tmplData := map[string]string{
		"signin_link": req.Token,
	}

	err := s.client.Send(ctx, req.Address, s.templateName, tmplData)

	return err
}
