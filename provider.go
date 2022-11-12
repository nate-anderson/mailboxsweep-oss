package mailsweeper

// Provider is an email service provider
type Provider interface {
	Connect() error
	Login(username, password string) (Inbox, error)
}
