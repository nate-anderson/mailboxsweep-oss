package login

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mailboxsweep/inboxuser"
	"time"
)

type LinkService interface {
	CreateLoginLink(ctx context.Context, user inboxuser.InboxUser, initiatedByIP string) error
	ValidateLoginLink(ctx context.Context, token, signature string) (userUUID string, err error)
}

type DefaultLinkService struct {
	repo         LinkRepository
	userService  inboxuser.Service
	sender       Sender
	linkLifespan time.Duration
	tokenLength  int
	signingCert  *rsa.PrivateKey
}

func NewLinkService(
	repo LinkRepository,
	userService inboxuser.Service,
	sender Sender,
	signingCert *rsa.PrivateKey,
	linkLifespan time.Duration,
	tokenLength int,
) *DefaultLinkService {
	return &DefaultLinkService{
		repo,
		userService,
		sender,
		linkLifespan,
		tokenLength,
		signingCert,
	}
}

func (s DefaultLinkService) signToken(tokenBytes []byte) (string, error) {
	checksum := sha256.Sum256(tokenBytes)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.signingCert, crypto.SHA256, checksum[:])
	if err != nil {
		return "", fmt.Errorf("failed to checksum token: %w", err)
	}

	sigString := hex.EncodeToString(signature)
	return sigString, nil
}

func (s DefaultLinkService) verifyToken(tokenBytes []byte, signature []byte) error {
	checksum := sha256.Sum256(tokenBytes)
	if err := rsa.VerifyPKCS1v15(&s.signingCert.PublicKey, crypto.SHA256, checksum[:], signature); err != nil {
		return fmt.Errorf("failed to validate signature of token: %w", err)
	}

	return nil
}

func (s DefaultLinkService) CreateLoginLink(ctx context.Context, user inboxuser.InboxUser, initiatedByIP string) error {

	tokenBytes := make([]byte, s.tokenLength)
	_, err := io.ReadFull(rand.Reader, tokenBytes)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	expires := now.Add(s.linkLifespan)
	token := hex.EncodeToString(tokenBytes)

	signature, err := s.signToken(tokenBytes)
	if err != nil {
		return err
	}

	link := Link{
		UserUUID:      user.UUID,
		Token:         token,
		Expires:       expires.Unix(),
		Created:       now,
		InitiatedByIP: initiatedByIP,
		Signature:     signature,
	}

	// only allow one active login link at a time
	if err = s.repo.InvalidateByUserUUID(ctx, user.UUID); err != nil {
		return err
	}

	if err = s.repo.Store(ctx, link); err != nil {
		return err
	}

	sendReq := SendRequest{
		Address:   user.Address,
		Expires:   expires,
		Token:     link.Token,
		Signature: link.Signature,
	}
	return s.sender.Send(ctx, sendReq)
}

func (s DefaultLinkService) ValidateLoginLink(ctx context.Context, token, signature string) (userUUID string, err error) {
	// verify signature before doing anything else
	tokenBytes, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}

	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return "", err
	}

	err = s.verifyToken(tokenBytes, signatureBytes)
	if err != nil {
		return "", fmt.Errorf("invalid login link (signature)")
	}

	link, err := s.repo.Find(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to find link by token: %w", err)
	}

	err = s.repo.InvalidateByUserUUID(ctx, link.UserUUID)
	if err != nil {
		return "", err
	}

	return link.UserUUID, nil
}
