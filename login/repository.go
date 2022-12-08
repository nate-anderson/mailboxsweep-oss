package login

import "context"

type LinkRepository interface {
	Store(ctx context.Context, link Link) error
	Find(ctx context.Context, token string) (*Link, error)
	InvalidateByUserUUID(ctx context.Context, uuid string) error
}
