package login

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"time"
)

type Service struct {
	repo         LinkRepository
	userService  inboxuser.Service
	sender       Sender
	linkLifespan time.Duration
	tokenLength  int
}

func NewService(repo LinkRepository, userService inboxuser.Service, sender Sender, linkLifespan time.Duration, tokenLength int) *Service {
	return &Service{
		repo,
		userService,
		sender,
		linkLifespan,
		tokenLength,
	}
}

func (s Service) CreateLoginLink(ctx context.Context, address, initiatedByIP string) error {
	user, err := s.userService.FindByAddress(ctx, address)
	if err != nil {
		return mailboxsweep.ErrNotFound
	}

	tokenBytes := make([]byte, s.tokenLength)
	_, err = io.ReadFull(rand.Reader, tokenBytes)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	expires := now.Add(s.linkLifespan)
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	link := Link{
		UserUUID:      user.UUID,
		Token:         token,
		Expires:       expires.Unix(),
		Created:       now,
		InitiatedByIP: initiatedByIP,
	}

	// only allow one active login link at a time
	if err = s.repo.InvalidateByUserUUID(ctx, user.UUID); err != nil {
		return err
	}

	if err = s.repo.Store(ctx, link); err != nil {
		return err
	}

	return s.sender.Send(ctx, SendRequest{
		Address: user.Address,
		Expires: expires,
		Token:   link.Token,
	})
}

func (s Service) ValidateLoginLink(ctx context.Context, token string) (*inboxuser.InboxUser, error) {
	link, err := s.repo.Find(ctx, token)
	if err != nil {
		return nil, err
	}

	err = s.repo.InvalidateByUserUUID(ctx, link.UserUUID)
	if err != nil {
		return nil, err
	}

	user, err := s.userService.FindByUUID(ctx, link.UserUUID)
	return user, err
}
