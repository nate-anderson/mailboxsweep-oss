package inbox

import "mailboxsweep/changeset"

// Provider is an email service provider
type Provider[messageIDType changeset.MessageID] interface {
	Connect() error
	Login(username, password string) (Inbox[messageIDType], error)
}
