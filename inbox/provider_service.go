package inbox

import (
	"context"
	"fmt"
	"mailboxsweep/imapserver"
	"net/mail"
	"strings"
)

type ProviderService struct {
	imapServerService imapserver.Service
}

func NewProviderService(imapServerService imapserver.Service) *ProviderService {
	return &ProviderService{
		imapServerService,
	}
}

func (s ProviderService) GetProvider(ctx context.Context, address string) (Provider, error) {
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

		provider, err = NewIMAPProvider(server.Host, server.Port)
		if err != nil {
			return nil, err
		}
	}

	return provider, nil
}
