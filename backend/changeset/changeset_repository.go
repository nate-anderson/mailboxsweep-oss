package changeset

import (
	"context"
)

type ChangeSetRepository interface {
	FindByInboxAndOperation(ctx context.Context, inbox string, operation UpdateOperation, consistentRead bool) (*ChangeSet, error)
	// create a deletion set for this inbox/folder pair if none exists
	// else update the messages in the deletion set
	// should return the number of items added by the operation
	Append(ctx context.Context, changeSet ChangeSet) ([]string, error)
	// should return the number of items removed by the operation
	Remove(ctx context.Context, changeset ChangeSet) ([]string, error)
	// Mark the changeset as executed. It will no longer be presented to the user.
	Archive(ctx context.Context, inbox string, operation UpdateOperation) error
}
