package inboxuser

import "context"

type Service interface {
	FindByAddress(ctx context.Context, address string) (*InboxUser, error)
	FindByUUID(ctx context.Context, uuid string) (*InboxUser, error)
}

type DefaultService struct {
	InboxUserRepository
}
