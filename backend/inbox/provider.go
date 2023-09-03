package inbox

import (
	"context"
	"errors"
	"mailboxsweep/inboxcredential"
)

// Provider is an email service provider
type Provider interface {
	Connect(ctx context.Context) error
	Login(ctx context.Context, username string, credential inboxcredential.Credential) (Inbox, error)

	// Decorate error with provider-specific info
	WrapError(err error) error
}

var (
	GmailError = errors.New("gmail error")
	IMAPError  = errors.New("imap error")
)
