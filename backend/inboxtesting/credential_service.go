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

	FindCredentialCalled int

	FindError error
}

func (t *TestCredentialReadService) FindCredential(ctx context.Context) (inboxcredential.Credential, error) {
	t.FindCredentialCalled++
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if t.FindError != nil {
		return nil, t.FindError
	}

	if address != t.Address {
		return nil, mailboxsweep.ErrNotFound
	}

	return &inboxcredential.IMAPInboxCredential{
		EmailAddress: address,
		Password:     t.Password,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}, nil
}

func (t TestCredentialReadService) DecryptCredential(ctx context.Context, cred inboxcredential.Credential) (inboxcredential.Credential, error) {
	return cred, nil
}
