package inboxtesting

import "mailboxsweep/inbox"

type TestProvider struct {
	Username string
	Password string

	ConnectError error
	LoginError   error
}

func (t TestProvider) Connect() error {
	return t.ConnectError
}

func (t *TestProvider) Login(username, password string) (inbox.Inbox, error) {
	t.Username = username
	t.Password = password

	if t.LoginError != nil {
		return nil, t.LoginError
	}

	return TestInbox{
		name: username,
	}, nil
}
