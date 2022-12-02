package inboxuser

import "context"

type InboxUserRepository interface {
	// store the inbox after successful creation
	// does not update, fails if the account already exists
	Store(ctx context.Context, user InboxUser) (uuid string, err error)

	FindByAddress(ctx context.Context, address string) (*InboxUser, error)
	FindByUUID(ctx context.Context, uuid string) (*InboxUser, error)

	StorePayment(ctx context.Context, address string, payment Payment) error

	Consent(ctx context.Context, address string, agreement Agreement) error

	TouchLastSeen(ctx context.Context, address string) error
}
