package login

import (
	"context"
	"mailboxsweep/email"
	"net/url"
	"time"
)

type EmailSender struct {
	client       email.Sender
	templateName string
	baseURL      string
}

func NewEmailSender(client email.Sender, templateName, baseURL string) *EmailSender {
	return &EmailSender{client, templateName, baseURL}
}

func (s EmailSender) Send(ctx context.Context, req SendRequest) error {
	query := url.Values{}
	query.Set("token", req.Token)
	query.Set("sig", req.Signature)
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()

	pst, err := time.LoadLocation("America/New_York")
	if err != nil {
		return err
	}
	tmplData := map[string]string{
		"signin_link": u.String(),
		"expiry_time": req.Expires.In(pst).Format("3:04 PM MST on Monday, Jan 2, 2006"),
	}

	err = s.client.Send(ctx, req.Address, s.templateName, tmplData)
	return err
}
