package inbox

import (
	"context"
	"fmt"
	"mailboxsweep/imapserver"
	"mailboxsweep/metering"
	"net/mail"
	"strings"
)

type ProviderService interface {
	GetProvider(ctx context.Context, address string) (Provider, error)
}

type DefaultProviderService struct {
	imapServerService imapserver.Service
	usageRepository   metering.UsageRepository
}

func NewProviderService(imapServerService imapserver.Service, usageRepository metering.UsageRepository) *DefaultProviderService {
	return &DefaultProviderService{
		imapServerService,
		usageRepository,
	}
}

// Gmail quota is 2500mb/day, allow IMAP clients other than us to keep working
var defaultQuota int = metering.Megabyte * 1800

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

		provider, err = NewIMAPProvider(ctx, server.Host, server.Port, address, s.usageRepository, &defaultQuota)
		if err != nil {
			return nil, err
		}
	}

	return provider, nil
}
