package inbox

import (
	"context"
	"mailboxsweep/inboxcredential"
)

// Provider is an email service provider
type Provider interface {
	Connect(ctx context.Context) error
	Login(ctx context.Context, username string, credential inboxcredential.Credential) (Inbox, error)
}
