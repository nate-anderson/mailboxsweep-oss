package changeset

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/classification"
)

type Service interface {
	MarkForDelete(ctx context.Context, folder string, paths []classification.Path) (int, error)
	MarkForMarkRead(ctx context.Context, folder string, paths []classification.Path) (int, error)
	UnmarkForDelete(ctx context.Context, folder string, paths []classification.Path) (int, error)
	UnmarkForMarkRead(ctx context.Context, folder string, paths []classification.Path) (int, error)
	ListByInbox(ctx context.Context, consistentRead bool) ([]ChangeSet, error)
	ListByInboxFolder(ctx context.Context, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error)
}

type DefaultService struct {
	repo ChangeSetRepository
}

func NewDefaultService(repo ChangeSetRepository) *DefaultService {
	return &DefaultService{repo}
}

func (s DefaultService) MarkForDelete(ctx context.Context, folder string, paths []classification.Path) (int, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return 0, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Folder:    folder,
		Operation: Delete,
		Paths:     pathStrings,
	}

	return s.repo.Append(ctx, cs)
}

func (s DefaultService) MarkForMarkRead(ctx context.Context, folder string, paths []classification.Path) (int, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return 0, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Folder:    folder,
		Operation: MarkRead,
		Paths:     pathStrings,
	}

	return s.repo.Append(ctx, cs)
}

func (s DefaultService) ListByInbox(ctx context.Context, consistentRead bool) ([]ChangeSet, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.ListByInbox(ctx, addr, consistentRead)
}

func (s DefaultService) ListByInboxFolder(ctx context.Context, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByInboxFolder(ctx, addr, folder, consistentRead)
}

func (s DefaultService) UnmarkForDelete(ctx context.Context, folder string, paths []classification.Path) (int, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return 0, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Folder:    folder,
		Operation: Delete,
		Paths:     pathStrings,
	}

	return s.repo.Remove(ctx, cs)
}

func (s DefaultService) UnmarkForMarkRead(ctx context.Context, folder string, paths []classification.Path) (int, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return 0, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Folder:    folder,
		Operation: MarkRead,
		Paths:     pathStrings,
	}

	return s.repo.Remove(ctx, cs)
}
