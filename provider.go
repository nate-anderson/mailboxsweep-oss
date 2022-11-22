package mailsweeper

// Provider is an email service provider
type Provider[messageIDType MessageID] interface {
	Connect() error
	Login(username, password string) (Inbox[messageIDType], error)
}
