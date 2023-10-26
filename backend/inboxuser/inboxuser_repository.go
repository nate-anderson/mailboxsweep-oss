package inboxuser

import "context"

type InboxUserRepository interface {
	// Store a *new* user
	// Note that an implementation of this method must fail if the user already exists
	// because it is difficult to efficiently check this further up the stack
	Store(ctx context.Context, user InboxUser) (uuid string, err error)

	FindByAddress(ctx context.Context, address string) (*InboxUser, error)
	FindByUUID(ctx context.Context, uuid string) (*InboxUser, error)

	StorePayment(ctx context.Context, address string, payment Payment) error

	SetName(ctx context.Context, address, nameFirst, nameLast string) error
	Consent(ctx context.Context, address string, agreement Agreement) error

	TouchLastSeen(ctx context.Context, address string) error
}
