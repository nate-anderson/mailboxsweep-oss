package inboxchangeset

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"

	"github.com/sirupsen/logrus"
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

	ClearSelection(ctx context.Context, operation changeset.UpdateOperation) error

	// do the deletes
	EnqueueCommitChanges(ctx context.Context, operation changeset.UpdateOperation) error
	CommitChanges(ctx context.Context, operation changeset.UpdateOperation) error
}

type DefaultService struct {
	changeSetService      changeset.Service
	classificationService classification.Service
	providerService       inbox.ProviderService
	credentialService     inboxcredential.CredentialReadService
	requestService        classifyrequest.Service
	metadataService       inbox.MetadataService
	commitQueuePublisher  mailboxsweep.EventPublisher[changeset.CommitQueueMessage]
	logger                *logrus.Logger
}

func NewService(
	changeSetService changeset.Service,
	classificationService classification.Service,
	providerService inbox.ProviderService,
	credentialService inboxcredential.CredentialReadService,
	requestService classifyrequest.Service,
	metadataService inbox.MetadataService,
	commitQueuePublisher mailboxsweep.EventPublisher[changeset.CommitQueueMessage],
	logger *logrus.Logger,
) *DefaultService {
	return &DefaultService{
		changeSetService:      changeSetService,
		classificationService: classificationService,
		providerService:       providerService,
		credentialService:     credentialService,
		requestService:        requestService,
		metadataService:       metadataService,
		commitQueuePublisher:  commitQueuePublisher,
		logger:                logger,
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

type updateFunc func(ctx context.Context, paths []classification.Path) ([]classification.Path, error)

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

	// store leaf message counts
	counts := make(map[string]int)
	pathsToProcess := []classification.Path{}

	for {
		select {
		case err := <-errChan:
			cancel()
			return nil, err
		case res := <-workChan:
			pathsToProcess = append(pathsToProcess, res.path)
			if increment {
				counts[res.path.String()] = res.diff
			} else {
				counts[res.path.String()] = res.diff * -1
			}
			if len(pathsToProcess) == len(paths) {
				goto Apply
			}

		}
	}

Apply:
	diffPaths, err := fn(groupCtx, pathsToProcess)
	if err != nil {
		return nil, err
	}

	for _, p := range diffPaths {
		diffCount, ok := counts[p.String()]
		if !ok {
			return nil, fmt.Errorf("path %s was found in applied diff but not stored in cached counts", p.String())
		}
		result.MsgCountDiff += diffCount
		result.Paths = append(result.Paths, p)
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

	log := s.logger.
		WithField("method", "CommitChanges").
		WithField("address", address).
		WithField("operation", string(operation))

	var msgChangeCount int

	defer func() {
		if err == nil {
			updateErr := s.requestService.MarkExecutedSuccess(ctx, msgChangeCount)
			if updateErr != nil {
				log.Error("failed to update request: ", updateErr)
			}
		} else if errors.Is(err, mailboxsweep.ErrGoogleReauthenticationRequired) {
			// the user will reauthenticate and can come back and submit this again
			log.Info("user needs to reauthenticate with google - changeset untouched")
		} else {
			updateErr := s.requestService.MarkExecutedError(ctx, msgChangeCount, err)
			if updateErr != nil {
				log.Error("failed to update request: ", updateErr)
			}

			// if something has gone wrong, invalidate inbox metadata
			invalidateErr := s.metadataService.Invalidate(ctx)
			if invalidateErr != nil {
				log.Error("failed to invalidate metadata after an error occurred: ", invalidateErr)
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
			if existing, ok := folderUIDs[m.Folder]; ok {
				folderUIDs[m.Folder] = append(existing, m.UID)
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
			err = folder.Delete(ctx, uids)
			if err != nil {
				return err
			}
		} else if operation == changeset.MarkRead {
			err = folder.MarkRead(ctx, uids)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown operation '%s'", operation)
		}

		msgChangeCount = len(uids)
	}

	err = s.updateInboxMetadata(ctx, diff, changeSet.Operation)
	if err != nil {
		return err
	}

	err = s.changeSetService.Archive(ctx, changeSet.Operation)
	if err != nil {
		return err
	}

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

		if operation == changeset.Delete {
			n += int(leaf.Count)
		} else {
			n += int(leaf.UnreadCount)
		}
	}

	return changeSet, n, nil
}

func (s DefaultService) ClearSelection(ctx context.Context, operation changeset.UpdateOperation) error {
	return s.changeSetService.ClearSelection(ctx, operation)
}

func (s DefaultService) EnqueueCommitChanges(ctx context.Context, operation changeset.UpdateOperation) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	if err = s.requestService.MarkExecuting(ctx, string(operation)); err != nil {
		return fmt.Errorf("failed to mark classify request as executing: %w", err)
	}

	return s.commitQueuePublisher.PublishEvent(ctx, mailboxsweep.EventBody[changeset.CommitQueueMessage]{
		Body: changeset.CommitQueueMessage{
			Inbox:     addr,
			Operation: operation,
		},
	})
}

func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
