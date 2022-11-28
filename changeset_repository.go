package mailboxsweep

import (
	"context"
)

type ChangeSetRepository[messageIDType MessageID] interface {
	ListByInbox(ctx context.Context, inbox string) ([]ChangeSet, error)
	FindByInboxFolder(ctx context.Context, inbox, folder string) (ChangeSet, error)
	// create a deletion set for this inbox/folder pair if none exists
	// else update the messages in the deletion set
	// should return the number of items added by the operation
	Append(ctx context.Context, changeSet ChangeSet) (int, error)
	// should return the number of items removed by the operation
	Remove(ctx context.Context, inbox, folder string) error
}

type UpdateOperation uint

const (
	MarkAsRead UpdateOperation = iota + 1
	Delete
)
