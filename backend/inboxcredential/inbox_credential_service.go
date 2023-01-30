package inboxcredential

import (
	"context"
	"errors"
	"mailboxsweep"
)

type CredentialReadService interface {
	FindCredential(ctx context.Context, address string) (*InboxCredential, error)
	DecryptCredential(ctx context.Context, cred *InboxCredential) (password string, err error)
}

type CredentialWriteService interface {
	StoreCredential(ctx context.Context, password string) (address string, err error)
	DeleteCredential(ctx context.Context, address string) error
}

type Service interface {
	CredentialReadService
	CredentialWriteService
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
	address, err = mailboxsweep.UserAddressFromContext(ctx)
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
	cred, err := s.repo.FindByAddress(ctx, address)
	if err != nil {
		if errors.Is(err, mailboxsweep.ErrNotFound) {
			return nil, mailboxsweep.ErrNoStoredCredential
		}
		return nil, err
	}
	return cred, nil
}

func (s DefaultService) DeleteCredential(ctx context.Context, address string) error {
	return s.repo.DeleteByAddress(ctx, address)
}

func (s DefaultService) DecryptCredential(ctx context.Context, cred *InboxCredential) (password string, err error) {
	pass, err := cred.decryptPassword(s.cipherKey)
	return pass, err
}
