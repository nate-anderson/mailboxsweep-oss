package mailsweeper

type Provider interface {
	Connect() error
	Login(username, password string) (Inbox, error)
}
