package inboxcredential

import "context"

type InboxCredentialRepository interface {
	Upsert(ctx context.Context, credential Credential) error
	FindByAddress(ctx context.Context, address string) (Credential, error)
	DeleteByAddress(ctx context.Context, address string) error
}
