package inboxcredential

import (
	"context"
	"mailboxsweep/session"
)

type Service interface {
	StoreCredential(ctx context.Context, password string) (address string, err error)
	FindCredential(ctx context.Context, address string) (*InboxCredential, error)
	DecryptCredential(ctx context.Context, cred *InboxCredential) (password string, err error)
	DeleteCredential(ctx context.Context, address string) error
}

type DefaultService struct {
	repo      InboxCredentialRepository
	cipherKey []byte
}

func NewService(repo InboxCredentialRepository, cipherKey []byte) Service {
	return &DefaultService{
		repo,
		cipherKey,
	}
}

func (s DefaultService) StoreCredential(ctx context.Context, password string) (address string, err error) {
	address, err = session.UserAddressFromContext(ctx)
	if err != nil {
		return "", err
	}
	credential, err := NewInboxCredential(address, password, s.cipherKey)
	if err != nil {
		return "", err
	}

	err = s.repo.Upsert(ctx, *credential)
	if err != nil {
		return "", err
	}

	return credential.Address, nil
}

func (s DefaultService) FindCredential(ctx context.Context, address string) (*InboxCredential, error) {
	return s.repo.FindByAddress(ctx, address)
}

func (s DefaultService) DeleteCredential(ctx context.Context, address string) error {
	return s.repo.DeleteByAddress(ctx, address)
}

func (s DefaultService) DecryptCredential(ctx context.Context, cred *InboxCredential) (password string, err error) {
	pass, err := cred.decryptPassword(s.cipherKey)
	return pass, err
}
