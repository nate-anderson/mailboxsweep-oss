package inboxchangeset

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"strconv"
)

type ChangeResult struct {
	// leaf Paths added or removed
	Paths []classification.Path
	// number of messages added or removed
	MsgCountDiff int
}

type Service interface {
	AddPathForDelete(ctx context.Context, path classification.Path) (*ChangeResult, error)
	AddPathForMarkRead(ctx context.Context, path classification.Path) (*ChangeResult, error)
	RemovePathForMarkRead(ctx context.Context, path classification.Path) (*ChangeResult, error)
	RemovePathForDelete(ctx context.Context, path classification.Path) (*ChangeResult, error)

	FindByOperation(ctx context.Context, operation changeset.UpdateOperation) (changeSet *changeset.ChangeSet, size int, err error)

	// do the deletes
	CommitChanges(ctx context.Context, operation changeset.UpdateOperation) error
}

type DefaultService struct {
	changeSetService      changeset.Service
	classificationService classification.Service
	providerService       inbox.ProviderService
	credentialService     inboxcredential.CredentialReadService
	requestService        classifyrequest.Service
	metadataService       inbox.MetadataService
}

func NewService(
	changeSetService changeset.Service,
	classificationService classification.Service,
	providerService inbox.ProviderService,
	credentialService inboxcredential.CredentialReadService,
	requestService classifyrequest.Service,
	metadataService inbox.MetadataService,
) *DefaultService {
	return &DefaultService{
		changeSetService:      changeSetService,
		classificationService: classificationService,
		providerService:       providerService,
		credentialService:     credentialService,
		requestService:        requestService,
		metadataService:       metadataService,
	}
}

/*
Note that the four methods returning *ChangeResult may return a partial result with an error
*/

func (s DefaultService) AddPathForDelete(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	result := ChangeResult{
		MsgCountDiff: 0,
	}

	for _, p := range leafPaths {
		leaf, err := s.classificationService.ClassificationLeaf(ctx, p)
		if err != nil {
			return &result, err
		}

		_, err = s.changeSetService.MarkForDelete(ctx, []classification.Path{p})
		if err != nil {
			return &result, err
		}

		result.MsgCountDiff += int(leaf.Count)
		result.Paths = append(result.Paths, p)
	}

	return &result, nil
}

func (s DefaultService) AddPathForMarkRead(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	result := ChangeResult{
		MsgCountDiff: 0,
	}

	for _, p := range leafPaths {
		leaf, err := s.classificationService.ClassificationLeaf(ctx, p)
		if err != nil {
			return &result, err
		}

		_, err = s.changeSetService.MarkForMarkRead(ctx, []classification.Path{p})
		if err != nil {
			return &result, err
		}
		result.MsgCountDiff += int(leaf.UnreadCount)
		result.Paths = append(result.Paths, p)
	}

	return &result, nil
}

func (s DefaultService) RemovePathForMarkRead(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	result := ChangeResult{}

	for _, p := range leafPaths {
		leaf, err := s.classificationService.ClassificationLeaf(ctx, p)
		if err != nil {
			return &result, err
		}

		_, err = s.changeSetService.UnmarkForMarkRead(ctx, []classification.Path{p})
		if err != nil {
			return &result, err
		}

		result.MsgCountDiff -= int(leaf.UnreadCount)
		result.Paths = append(result.Paths, p)
	}

	return &result, nil
}

func (s DefaultService) RemovePathForDelete(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	result := ChangeResult{}

	for _, p := range leafPaths {
		leaf, err := s.classificationService.ClassificationLeaf(ctx, p)
		if err != nil {
			return &result, err
		}

		_, err = s.changeSetService.UnmarkForDelete(ctx, []classification.Path{path})
		if err != nil {
			return &result, err
		}

		result.MsgCountDiff -= int(leaf.Count)
		result.Paths = append(result.Paths, p)
	}

	return &result, nil
}

// used for tracking changes to inbox counts
type messageCountDiff struct {
	markedReadCount    int
	unreadDeletedCount int
	readDeletedCount   int
}

func (s DefaultService) CommitChanges(ctx context.Context, operation changeset.UpdateOperation) error {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			updateErr := s.requestService.MarkExecutedSuccess(ctx)
			if updateErr != nil {
				log.Println("failed to update request: ", updateErr)
			}
		} else {
			updateErr := s.requestService.MarkExecutedError(ctx, err)
			if updateErr != nil {
				log.Println("failed to update request: ", updateErr)
			}

			// if something has gone wrong, invalidate inbox metadata
			invalidateErr := s.metadataService.Invalidate(ctx)
			if invalidateErr != nil {
				log.Println("failed to invalidate metadata after an error occurred: ", invalidateErr)
			}
		}
	}()

	changeSet, err := s.changeSetService.FindByOperation(ctx, operation, true)
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

	var diff messageCountDiff

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

			// track changes to inbox counts
			if m.Read && operation == changeset.Delete {
				diff.readDeletedCount++
			} else if !m.Read && operation == changeset.Delete {
				diff.unreadDeletedCount++
			} else if !m.Read && operation == changeset.MarkRead {
				diff.markedReadCount++
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

	err = s.updateInboxMetadata(ctx, diff, changeSet.Operation)
	return err
}

func (s DefaultService) updateInboxMetadata(ctx context.Context, diff messageCountDiff, operation changeset.UpdateOperation) error {
	meta, err := s.metadataService.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to update inbox metadata: %w", err)
	}

	// unread count shrinks by the number of newly marked read messages
	meta.UnreadCount -= diff.markedReadCount
	// unread coutn shrinks by number of unread messages deleted
	meta.UnreadCount -= diff.unreadDeletedCount
	// total messages shrinks by number of messages deleted
	meta.MessageCount -= diff.readDeletedCount
	meta.MessageCount -= diff.unreadDeletedCount

	err = s.metadataService.Set(ctx, *meta)
	return err
}

func (s DefaultService) FindByOperation(ctx context.Context, operation changeset.UpdateOperation) (*changeset.ChangeSet, int, error) {
	changeSet, err := s.changeSetService.FindByOperation(ctx, operation, false)
	if err != nil {
		return nil, 0, err
	}

	n := 0

	for _, path := range changeSet.Paths {
		leaf, err := s.classificationService.ClassificationLeaf(ctx, classification.ParsePath(path))
		if err != nil {
			return nil, 0, err
		}

		n += int(leaf.Count)
	}

	return changeSet, n, nil
}
