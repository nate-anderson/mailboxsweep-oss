package inboxchangeset

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"strconv"
)

type Service interface {
	AddLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error)
	AddLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error)
	RemoveLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error)
	RemoveLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error)

	// do the deletes
	CommitChanges(ctx context.Context) error
}

type DefaultService struct {
	changeSetService      changeset.Service
	classificationService classification.Service
	providerService       inbox.ProviderService
	credentialService     inboxcredential.CredentialReadService
}

func NewService(
	changeSetService changeset.Service,
	classificationService classification.Service,
	providerService inbox.ProviderService,
	credentialService inboxcredential.CredentialReadService,
) *DefaultService {
	return &DefaultService{
		changeSetService:      changeSetService,
		classificationService: classificationService,
		providerService:       providerService,
		credentialService:     credentialService,
	}
}

func (s DefaultService) AddLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.MarkForDelete(ctx, folder, []classification.Path{path})
}

func (s DefaultService) AddLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.MarkForMarkRead(ctx, folder, []classification.Path{path})
}

func (s DefaultService) RemoveLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.UnmarkForMarkRead(ctx, folder, []classification.Path{path})
}

func (s DefaultService) RemoveLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.UnmarkForDelete(ctx, folder, []classification.Path{path})
}

func (s DefaultService) CommitChanges(ctx context.Context) error {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	changeSets, err := s.changeSetService.ListByInbox(ctx, true)
	if err != nil {
		return err
	}

	provider, err := s.providerService.GetProvider(ctx, address)
	if err != nil {
		return err
	}

	cred, err := s.credentialService.FindCredential(ctx, address)
	if err != nil {
		return err
	}

	password, err := s.credentialService.DecryptCredential(ctx, cred)
	if err != nil {
		return err
	}

	mailbox, err := provider.Login(address, password)
	if err != nil {
		return err
	}

	for _, cs := range changeSets {
		folder, err := mailbox.Folder(cs.Folder)
		if err != nil {
			return err
		}

		for _, pathStr := range cs.Paths {
			path := classification.ParsePath(pathStr)
			leaf, err := s.classificationService.ClassificationLeaf(ctx, path)
			if err != nil {
				return err
			}

			uids := make([]string, len(leaf.Messages))
			for i, m := range leaf.Messages {
				uids[i] = strconv.FormatUint(uint64(m.UID), 10)
			}

			switch cs.Operation {
			case changeset.Delete:
				if err = folder.Delete(uids); err != nil {
					return err
				}
				break
			case changeset.MarkRead:
				if err = folder.MarkRead(uids); err != nil {
					return err
				}
				break
			default:
				return fmt.Errorf("unknown change operation %s", cs.Operation)
			}

		}
	}

	return nil
}
