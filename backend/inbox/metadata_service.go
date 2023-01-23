package inbox

import (
	"context"
	"errors"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"time"
)

type MetadataService interface {
	Get(ctx context.Context) (*Metadata, error)
	Set(ctx context.Context, m Metadata) error
}

type DefaultMetadataService struct {
	repo              MetadataRepository
	providerService   ProviderService
	credentialService inboxcredential.CredentialReadService
	recordTTL         time.Duration
}

func NewMetadataService(
	repo MetadataRepository,
	providerService ProviderService,
	credentialService inboxcredential.CredentialReadService,
	recordTTL time.Duration,
) *DefaultMetadataService {
	return &DefaultMetadataService{
		repo,
		providerService,
		credentialService,
		recordTTL,
	}
}

func (s DefaultMetadataService) Get(ctx context.Context) (*Metadata, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// check cache
	m, err := s.repo.Get(ctx, addr)
	if err == nil {
		return m, nil
	}

	if !errors.Is(err, mailboxsweep.ErrNotFound) {
		return nil, err
	}

	// cache miss
	provider, err := s.providerService.GetProvider(ctx, addr)
	if err != nil {
		return nil, err
	}

	credential, err := s.credentialService.FindCredential(ctx, addr)
	if err != nil {
		return nil, err
	}

	password, err := s.credentialService.DecryptCredential(ctx, credential)
	if err != nil {
		return nil, err
	}

	mailbox, err := provider.Login(addr, password)
	if err != nil {
		return nil, err
	}

	folders, err := mailbox.Folders()
	if err != nil {
		return nil, err
	}

	m = &Metadata{
		Inbox:   addr,
		Folders: make(map[string]uint32),
		Expires: time.Now().Add(s.recordTTL).Unix(),
	}
	for _, name := range folders {
		folder, err := mailbox.Folder(name)
		if err != nil {
			return nil, err
		}
		m.Folders[name] = folder.Count()
	}

	// write to cache
	err = s.repo.Store(ctx, *m)
	return m, err
}

func (s DefaultMetadataService) Set(ctx context.Context, m Metadata) error {
	if addr, err := mailboxsweep.UserAddressFromContext(ctx); err != nil {
		return err
	} else if addr != m.Inbox {
		return errors.New("metadata address does not match authenticated user")
	}

	return s.repo.Store(ctx, m)
}
