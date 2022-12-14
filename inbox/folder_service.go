package inbox

import (
	"context"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/session"
)

type FolderService interface {
	ListInboxFolders(ctx context.Context) ([]string, error)
}

type DefaultFolderService struct {
	providerService   ProviderService
	credentialService inboxcredential.Service
}

func NewFolderService(providerService ProviderService, credentialService inboxcredential.Service) FolderService {
	return &DefaultFolderService{
		providerService:   providerService,
		credentialService: credentialService,
	}
}

func (s DefaultFolderService) ListInboxFolders(ctx context.Context) ([]string, error) {
	address, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	provider, err := s.providerService.GetProvider(ctx, address)
	if err != nil {
		return nil, err
	}

	cred, err := s.credentialService.FindCredential(ctx, address)
	if err != nil {
		return nil, err
	}

	password, err := s.credentialService.DecryptCredential(ctx, cred)
	if err != nil {
		return nil, err
	}

	inbox, err := provider.Login(address, password)
	if err != nil {
		return nil, err
	}

	folders, err := inbox.Folders()
	return folders, err
}
