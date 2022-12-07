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
	linkLifespan time.Duration
	tokenLength  int
}

func (s Service) CreateLoginLink(ctx context.Context, address, initiatedByIP string) error {
	user, err := s.userService.FindByUUID(ctx, address)
	if err != nil {
		return mailboxsweep.ErrNotFound
	}

	tokenBytes := make([]byte, s.tokenLength)
	_, err = io.ReadFull(rand.Reader, tokenBytes)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	link := Link{
		UserUUID:      user.UUID,
		Token:         token,
		Expires:       now.Add(s.linkLifespan).Unix(),
		Created:       now,
		InitiatedByIP: initiatedByIP,
	}

	// only allow one active login link at a time
	if err = s.repo.InvalidateByAddress(ctx, address); err != nil {
		return err
	}

	return s.repo.Store(ctx, link)
}
