package inboxuser

import (
	"context"
	"errors"
	"mailboxsweep/session"
	"strings"

	"github.com/mcnijman/go-emailaddress"
)

type Service interface {
	Store(ctx context.Context, address string) (*InboxUser, error)
	FindByAddress(ctx context.Context, address string) (*InboxUser, error)
	FindByUUID(ctx context.Context, uuid string) (*InboxUser, error)
	SetName(ctx context.Context, nameFirst, nameLast string) error
	Consent(ctx context.Context, agreements []Agreement) error
}

func NewService(repo InboxUserRepository) DefaultService {
	return DefaultService{
		repo: repo,
	}
}

type DefaultService struct {
	repo InboxUserRepository
}

func (s DefaultService) Store(ctx context.Context, address string) (*InboxUser, error) {
	_, err := emailaddress.Parse(address)
	if err != nil {
		return nil, err
	}

	user, err := NewInboxUser(strings.TrimSpace(address))
	if err != nil {
		return nil, err
	}

	if _, err = s.repo.Store(ctx, *user); err != nil {
		return nil, err
	}

	return s.repo.FindByAddress(ctx, address)
}

func (s DefaultService) Consent(ctx context.Context, agreements []Agreement) error {
	userAddress, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.repo.FindByAddress(ctx, userAddress)
	if err != nil {
		return err
	}

	for _, agreement := range agreements {
		err = s.repo.Consent(ctx, user.Address, agreement)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s DefaultService) FindByAddress(ctx context.Context, address string) (*InboxUser, error) {
	return s.repo.FindByAddress(ctx, address)
}

func (s DefaultService) FindByUUID(ctx context.Context, uuid string) (*InboxUser, error) {
	return s.repo.FindByUUID(ctx, uuid)
}

func (s DefaultService) SetName(ctx context.Context, nameFirst, nameLast string) error {
	if strings.TrimSpace(nameFirst) == "" {
		return errors.New("first name is required")
	}

	if strings.TrimSpace(nameLast) == "" {
		return errors.New("last name is required")
	}

	userAddress, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.repo.FindByAddress(ctx, userAddress)
	if err != nil {
		return err
	}

	return s.repo.SetName(ctx, user.Address, nameFirst, nameLast)
}
