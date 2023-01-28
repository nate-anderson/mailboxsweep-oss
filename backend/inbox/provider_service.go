package inbox

import (
	"context"
	"fmt"
	"mailboxsweep/imapserver"
	"net/mail"
	"strings"
)

type ProviderService interface {
	GetProvider(ctx context.Context, address string) (Provider, error)
}

type DefaultProviderService struct {
	imapServerService imapserver.Service
}

func NewProviderService(imapServerService imapserver.Service) *DefaultProviderService {
	return &DefaultProviderService{
		imapServerService,
	}
}

func (s DefaultProviderService) GetProvider(ctx context.Context, address string) (Provider, error) {
	_, err := mail.ParseAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	tokens := strings.Split(address, "@")
	lastIdx := len(tokens) - 1
	domain := strings.ToLower(tokens[lastIdx])

	var provider Provider
	switch domain {

	// default to IMAP
	default:
		server, err := s.imapServerService.FindByDomain(ctx, domain)
		if err != nil {
			return nil, err
		}

		provider, err = NewIMAPProvider(server.Host, server.Port, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	return provider, nil
}
