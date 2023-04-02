package inbox

import "mailboxsweep/inboxcredential"

// Provider is an email service provider
type Provider interface {
	Connect() error
	Login(username string, credential inboxcredential.Credential) (Inbox, error)
}
