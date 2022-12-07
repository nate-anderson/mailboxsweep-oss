package login

import "context"

type LinkRepository interface {
	Store(ctx context.Context, link Link) error
	Find(ctx context.Context, token string) (*Link, error)
	InvalidateByAddress(ctx context.Context, address string) error
}
