package login

import (
	"context"
	"log"
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
	urlStr := u.String()
	tmplData := map[string]string{
		"signin_link": urlStr,
		"expiry_time": req.Expires.In(pst).Format("3:04 PM MST on Monday, Jan 2, 2006"),
	}

	// @TODO: Remove this
	log.Printf("Login link for %s: %s", req.Address, urlStr)

	err = s.client.Send(ctx, req.Address, s.templateName, tmplData)
	return err
}
