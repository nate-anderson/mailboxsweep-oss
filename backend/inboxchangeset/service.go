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

	"golang.org/x/exp/constraints"
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

// used for concurrency
type pathChangeResult struct {
	path classification.Path
	diff int
}

/*
Note that the four methods returning *ChangeResult may return a partial result with an error
*/

type updateFunc func(ctx context.Context, paths []classification.Path) (int, error)

func (s DefaultService) applyToManyPaths(ctx context.Context, paths []classification.Path, fn updateFunc, increment, useUnreadCount bool) (*ChangeResult, error) {
	var (
		result           = ChangeResult{}
		errChan          = make(chan error)
		workChan         = make(chan pathChangeResult)
		groupCtx, cancel = context.WithCancel(ctx)
	)

	defer cancel()

	go func() {
		for _, p := range paths {
			go func(leafPath classification.Path) {
				// confirm leaf exists, get diff
				leaf, err := s.classificationService.ClassificationLeaf(groupCtx, leafPath)
				if err != nil {
					errChan <- err
					return
				}

				res := pathChangeResult{
					path: leafPath,
				}
				if useUnreadCount {
					res.diff = int(leaf.UnreadCount)
				} else {
					res.diff = int(leaf.Count)
				}

				workChan <- res
			}(p)
		}
	}()

	for {
		select {
		case err := <-errChan:
			cancel()
			return nil, err
		case res := <-workChan:
			result.Paths = append(result.Paths, res.path)
			if increment {
				result.MsgCountDiff += res.diff
			} else {
				result.MsgCountDiff -= res.diff
			}
			if len(result.Paths) == len(paths) {
				goto Apply
			}

		}
	}

Apply:
	_, err := fn(groupCtx, result.Paths)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s DefaultService) AddPathForDelete(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	return s.applyToManyPaths(ctx, leafPaths, s.changeSetService.MarkForDelete, true, false)
}

func (s DefaultService) AddPathForMarkRead(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	return s.applyToManyPaths(ctx, leafPaths, s.changeSetService.MarkForMarkRead, true, true)
}

func (s DefaultService) RemovePathForMarkRead(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	return s.applyToManyPaths(ctx, leafPaths, s.changeSetService.UnmarkForMarkRead, false, true)
}

func (s DefaultService) RemovePathForDelete(ctx context.Context, path classification.Path) (*ChangeResult, error) {
	leafPaths, err := s.classificationService.GetLeavesOfPrefix(ctx, path)
	if err != nil {
		return nil, err
	}

	return s.applyToManyPaths(ctx, leafPaths, s.changeSetService.UnmarkForDelete, false, false)
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

	var msgChangeCount int

	defer func() {
		if err == nil {
			updateErr := s.requestService.MarkExecutedSuccess(ctx, string(operation), msgChangeCount)
			if updateErr != nil {
				log.Println("failed to update request: ", updateErr)
			}
		} else {
			updateErr := s.requestService.MarkExecutedError(ctx, string(operation), msgChangeCount, err)
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

	mailbox, err := provider.Login(ctx, address, password)
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
				folderUIDs[m.Folder] = append(folderUIDs[m.Folder], m.UID)
			} else {
				folderUIDs[m.Folder] = []string{m.UID}
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

		msgChangeCount = len(uids)
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
	meta.UnreadCount = max(meta.UnreadCount-diff.markedReadCount, 0)
	// unread coutn shrinks by number of unread messages deleted
	meta.UnreadCount = max(meta.UnreadCount-diff.unreadDeletedCount, 0)
	// total messages shrinks by number of messages deleted
	meta.MessageCount = max(meta.MessageCount-diff.readDeletedCount, 0)
	meta.MessageCount = max(meta.MessageCount-diff.unreadDeletedCount, 0)

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

func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
