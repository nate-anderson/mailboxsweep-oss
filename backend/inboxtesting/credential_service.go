package inboxtesting

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"time"
)

type TestCredentialReadService struct {
	Address  string
	Password string

	FindError error
}

func (t TestCredentialReadService) FindCredential(ctx context.Context, address string) (*inboxcredential.InboxCredential, error) {
	if t.FindError != nil {
		return nil, t.FindError
	}

	if address != t.Address {
		return nil, mailboxsweep.ErrNotFound
	}

	return &inboxcredential.InboxCredential{
		Address:  address,
		Password: t.Password,
		Created:  time.Now(),
		Expires:  time.Now().Add(time.Hour).Unix(),
	}, nil
}

func (t TestCredentialReadService) DecryptCredential(ctx context.Context, cred *inboxcredential.InboxCredential) (password string, err error) {
	return cred.Password, nil
}
