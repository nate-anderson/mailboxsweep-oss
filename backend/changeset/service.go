package changeset

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/classification"
)

type Service interface {
	MarkForDelete(ctx context.Context, paths []classification.Path) (added []classification.Path, err error)
	MarkForMarkRead(ctx context.Context, paths []classification.Path) (added []classification.Path, err error)
	UnmarkForDelete(ctx context.Context, paths []classification.Path) (removed []classification.Path, err error)
	UnmarkForMarkRead(ctx context.Context, paths []classification.Path) (removed []classification.Path, err error)
	FindByOperation(ctx context.Context, operation UpdateOperation, consistentRead bool) (*ChangeSet, error)
	Archive(ctx context.Context, operation UpdateOperation) error
}

type DefaultService struct {
	repo ChangeSetRepository
}

func NewDefaultService(repo ChangeSetRepository) *DefaultService {
	return &DefaultService{repo}
}

func (s DefaultService) MarkForDelete(ctx context.Context, paths []classification.Path) ([]classification.Path, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Operation: Delete,
		Paths:     pathStrings,
	}

	rawPaths, err := s.repo.Append(ctx, cs)
	if err != nil {
		return nil, err
	}

	addedPaths := make([]classification.Path, len(rawPaths))
	for i, raw := range rawPaths {
		addedPaths[i] = classification.ParsePath(raw)
	}
	return addedPaths, nil
}

func (s DefaultService) MarkForMarkRead(ctx context.Context, paths []classification.Path) ([]classification.Path, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Operation: MarkRead,
		Paths:     pathStrings,
	}

	rawPaths, err := s.repo.Append(ctx, cs)

	addedPaths := make([]classification.Path, len(rawPaths))
	for i, raw := range rawPaths {
		addedPaths[i] = classification.ParsePath(raw)
	}
	return addedPaths, nil
}

func (s DefaultService) FindByOperation(ctx context.Context, operation UpdateOperation, consistentRead bool) (*ChangeSet, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByInboxAndOperation(ctx, addr, operation, consistentRead)
}

func (s DefaultService) UnmarkForDelete(ctx context.Context, paths []classification.Path) ([]classification.Path, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Operation: Delete,
		Paths:     pathStrings,
	}

	rawPaths, err := s.repo.Remove(ctx, cs)
	if err != nil {
		return nil, err
	}

	removedPaths := make([]classification.Path, len(rawPaths))
	for i, raw := range rawPaths {
		removedPaths[i] = classification.ParsePath(raw)
	}
	return removedPaths, nil
}

func (s DefaultService) UnmarkForMarkRead(ctx context.Context, paths []classification.Path) ([]classification.Path, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	pathStrings := make([]string, len(paths))
	for i, p := range paths {
		pathStrings[i] = p.String()
	}

	cs := ChangeSet{
		Inbox:     addr,
		Operation: MarkRead,
		Paths:     pathStrings,
	}

	rawPaths, err := s.repo.Remove(ctx, cs)
	if err != nil {
		return nil, err
	}

	removedPaths := make([]classification.Path, len(rawPaths))
	for i, raw := range rawPaths {
		removedPaths[i] = classification.ParsePath(raw)
	}
	return removedPaths, nil
}

func (s DefaultService) Archive(ctx context.Context, operation UpdateOperation) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.Archive(ctx, addr, operation)
}
