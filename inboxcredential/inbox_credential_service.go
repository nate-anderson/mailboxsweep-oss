package inboxcredential

import "context"

type CreateCredentialInput struct {
	Address  string
	Password string
}

type Service interface {
	StoreCredential(ctx context.Context, input CreateCredentialInput) (address string, err error)
	FindCredential(ctx context.Context, address string) (*InboxCredential, error)
	DecryptCredential(ctx context.Context, cred *InboxCredential) (password string, err error)
	DeleteCredential(ctx context.Context, address string) error
}

type DefaultService struct {
	repo      InboxCredentialRepository
	cipherKey []byte
}

func (s DefaultService) StoreCredential(ctx context.Context, input CreateCredentialInput) (address string, err error) {
	credential, err := NewInboxCredential(input.Address, input.Password, s.cipherKey)
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
