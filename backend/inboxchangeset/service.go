package inboxchangeset

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"strconv"
)

type Service interface {
	AddLeafForDelete(ctx context.Context, path classification.Path) (int, error)
	AddLeafForMarkRead(ctx context.Context, path classification.Path) (int, error)
	RemoveLeafForMarkRead(ctx context.Context, path classification.Path) (int, error)
	RemoveLeafForDelete(ctx context.Context, path classification.Path) (int, error)

	// do the deletes
	CommitChanges(ctx context.Context, operation changeset.UpdateOperation) error
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

func (s DefaultService) AddLeafForDelete(ctx context.Context, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.MarkForDelete(ctx, []classification.Path{path})
}

func (s DefaultService) AddLeafForMarkRead(ctx context.Context, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.MarkForMarkRead(ctx, []classification.Path{path})
}

func (s DefaultService) RemoveLeafForMarkRead(ctx context.Context, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.UnmarkForMarkRead(ctx, []classification.Path{path})
}

func (s DefaultService) RemoveLeafForDelete(ctx context.Context, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.UnmarkForDelete(ctx, []classification.Path{path})
}

func (s DefaultService) CommitChanges(ctx context.Context, operation changeset.UpdateOperation) error {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	changeSet, err := s.changeSetService.FindByInboxAndOperation(ctx, operation, true)
	if err != nil {
		return err
	}
	if changeSet.Operation != operation {
		return errors.New("retrieved operation does not match requested")
	}

	provider, err := s.providerService.GetProvider(ctx, address)
	if err != nil {
		return err
	}

	cred, err := s.credentialService.FindCredential(ctx)
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

	// group UIDs by folder for more efficient delete
	folderUIDs := make(map[string][]string)

	for _, pathStr := range changeSet.Paths {
		path := classification.ParsePath(pathStr)
		leaf, err := s.classificationService.ClassificationLeaf(ctx, path)
		if err != nil {
			return err
		}

		for _, m := range leaf.Messages {
			if _, ok := folderUIDs[m.Folder]; ok {
				folderUIDs[m.Folder] = append(folderUIDs[m.Folder], strconv.FormatUint(uint64(m.UID), 10))
			} else {
				folderUIDs[m.Folder] = []string{strconv.FormatUint(uint64(m.UID), 10)}
			}
		}
	}

	for folderName, uids := range folderUIDs {
		folder, err := mailbox.Folder(folderName)
		if err != nil {
			return err
		}

		if operation == changeset.Delete {
			err = folder.Delete(uids)
			if err != nil {
				return err
			}
		} else if operation == changeset.MarkRead {
			err = folder.MarkRead(uids)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown operation '%s'", operation)
		}

	}

	return nil
}
