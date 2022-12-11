package login

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"time"
)

type Service interface {
	CreateLoginLink(ctx context.Context, address, initiatedByIP string) (existingUser bool, error error)
	ValidateLoginLink(ctx context.Context, token string) (*inboxuser.InboxUser, error)
}

type DefaultService struct {
	repo         LinkRepository
	userService  inboxuser.Service
	sender       Sender
	linkLifespan time.Duration
	tokenLength  int
}

func NewService(repo LinkRepository, userService inboxuser.Service, sender Sender, linkLifespan time.Duration, tokenLength int) *DefaultService {
	return &DefaultService{
		repo,
		userService,
		sender,
		linkLifespan,
		tokenLength,
	}
}

func (s DefaultService) CreateLoginLink(ctx context.Context, address, initiatedByIP string) (existingUser bool, error error) {
	user, err := s.userService.FindByAddress(ctx, address)
	sendAddress := address
	if err != nil {
		// if this is an error other than "not found", fail
		if !errors.Is(err, mailboxsweep.ErrNotFound) {
			return false, mailboxsweep.ErrNotFound
		}

		// store the new user
		user, err = s.userService.Store(ctx, address)
		if err != nil {
			return false, err
		}

		// otherwise, consider this first login
	} else {
		sendAddress = user.Address
		existingUser = true
	}

	tokenBytes := make([]byte, s.tokenLength)
	_, err = io.ReadFull(rand.Reader, tokenBytes)
	if err != nil {
		return false, fmt.Errorf("failed to generate token: %w", err)
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
		return false, err
	}

	if err = s.repo.Store(ctx, link); err != nil {
		return false, err
	}

	sendReq := SendRequest{
		Address: sendAddress,
		Expires: expires,
		Token:   link.Token,
	}
	return existingUser, s.sender.Send(ctx, sendReq)
}

func (s DefaultService) ValidateLoginLink(ctx context.Context, token string) (*inboxuser.InboxUser, error) {
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
