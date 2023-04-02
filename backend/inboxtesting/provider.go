package inboxtesting

import (
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
)

type TestProvider struct {
	Username string
	Password string

	Messages map[string][]inbox.Message

	ConnectError error
	LoginError   error

	// maintain reference for testing
	// @TODO: figure out a better way to mock than this
	// one inbox per username
	Inboxes map[string]*TestInbox
}

func (t TestProvider) Connect() error {
	return t.ConnectError
}

func (t *TestProvider) Login(username string, credential inboxcredential.Credential) (inbox.Inbox, error) {
	t.Username = username
	if imapCred, ok := credential.(*inboxcredential.IMAPInboxCredential); ok {
		t.Password = imapCred.Password
	}

	if t.LoginError != nil {
		return nil, t.LoginError
	}

	if existing, ok := t.Inboxes[username]; ok {
		return existing, nil
	}

	inbox := &TestInbox{
		name:         username,
		messages:     t.Messages,
		InboxFolders: make(map[string]*TestFolder),
	}

	t.Inboxes[username] = inbox
	return inbox, nil
}
