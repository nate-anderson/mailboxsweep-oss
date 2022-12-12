package changeset

import (
	"context"
	"mailboxsweep/session"
)

type Service interface {
	MarkForDelete(ctx context.Context, folder string, uids []string) (int, error)
	MarkForMarkRead(ctx context.Context, folder string, uids []string) (int, error)
	ListByInbox(ctx context.Context, folder string, consistentRead bool) ([]ChangeSet, error)
	ListByInboxFolder(ctx context.Context, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error)
}

type DefaultService struct {
	repo ChangeSetRepository
}

func NewDefaultService(repo ChangeSetRepository) *DefaultService {
	return &DefaultService{repo}
}

func (s DefaultService) MarkForDelete(ctx context.Context, folder string, uids []string) (int, error) {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return 0, err
	}

	cs := ChangeSet{
		Inbox:     addr,
		Folder:    folder,
		Operation: Delete,
		UIDs:      uids,
	}

	return s.repo.Append(ctx, cs)
}

func (s DefaultService) MarkForMarkRead(ctx context.Context, folder string, uids []string) (int, error) {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return 0, err
	}

	cs := ChangeSet{
		Inbox:     addr,
		Folder:    folder,
		Operation: MarkRead,
		UIDs:      uids,
	}

	return s.repo.Append(ctx, cs)
}

func (s DefaultService) ListByInbox(ctx context.Context, consistentRead bool) ([]ChangeSet, error) {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.ListByInbox(ctx, addr, consistentRead)
}

func (s DefaultService) ListByInboxFolder(ctx context.Context, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error) {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByInboxFolder(ctx, addr, folder, consistentRead)
}
