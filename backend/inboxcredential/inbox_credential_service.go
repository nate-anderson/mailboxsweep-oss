package inboxcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mailboxsweep"
)

type CredentialReadService interface {
	FindCredential(ctx context.Context) (Credential, error)
	DecryptCredential(ctx context.Context, cred Credential) (Credential, error)
}

type CredentialWriteService interface {
	StoreCredential(ctx context.Context, cred Credential) error
	DeleteCredential(ctx context.Context) error
}

type Service interface {
	CredentialReadService
	CredentialWriteService
}

type DefaultService struct {
	repo      InboxCredentialRepository
	cipherKey []byte
	encryptor cipher.AEAD
}

func NewService(repo InboxCredentialRepository, cipherKey []byte) (*DefaultService, error) {
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	encryptor, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher AEAD: %w", err)
	}

	return &DefaultService{
		repo,
		cipherKey,
		encryptor,
	}, nil
}

// StoreCredential encrypts then stores the credential
func (s DefaultService) StoreCredential(ctx context.Context, credential Credential) (err error) {
	_, err = mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	if err = credential.encrypt(s.encryptor); err != nil {
		return err
	}

	err = s.repo.Upsert(ctx, credential)
	if err != nil {
		return err
	}

	return nil
}

// FindCredential retrieves and returns the encrypted credential. Call `DecryptCredential` to get plaintext
func (s DefaultService) FindCredential(ctx context.Context) (Credential, error) {
	var cred Credential
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return cred, err
	}
	cred, err = s.repo.FindByAddress(ctx, address)
	if err != nil {
		if errors.Is(err, mailboxsweep.ErrNotFound) {
			return cred, mailboxsweep.NoStoredCredentialError(err)
		}
		return cred, err
	}

	return cred, nil
}

func (s DefaultService) DeleteCredential(ctx context.Context) error {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	return s.repo.DeleteByAddress(ctx, address)
}

func (s DefaultService) DecryptCredential(ctx context.Context, cred Credential) (credential Credential, err error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = cred.decrypt(s.encryptor)
	if cred.Address() != addr {
		return nil, errors.New("incorrect context user for credential")
	}
	return cred, err
}
