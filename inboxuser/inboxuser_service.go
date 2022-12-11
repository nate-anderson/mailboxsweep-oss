package inboxuser

import (
	"context"

	"github.com/mcnijman/go-emailaddress"
)

type Service interface {
	Store(ctx context.Context, address string) (*InboxUser, error)
	FindByAddress(ctx context.Context, address string) (*InboxUser, error)
	FindByUUID(ctx context.Context, uuid string) (*InboxUser, error)
}

func NewService(repo InboxUserRepository) Service {
	return DefaultService{
		InboxUserRepository: repo,
	}
}

type DefaultService struct {
	InboxUserRepository
}

func (s DefaultService) Store(ctx context.Context, address string) (*InboxUser, error) {
	_, err := emailaddress.Parse(address)
	if err != nil {
		return nil, err
	}

	user, err := NewInboxUser(address)
	if err != nil {
		return nil, err
	}

	if _, err = s.InboxUserRepository.Store(ctx, *user); err != nil {
		return nil, err
	}

	return s.InboxUserRepository.FindByAddress(ctx, address)
}
