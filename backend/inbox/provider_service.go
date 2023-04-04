package inbox

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/imapserver"
	"mailboxsweep/metering"
)

type ProviderService interface {
	GetProvider(ctx context.Context, address string) (Provider, error)
}

type DefaultProviderService struct {
	imapServerService imapserver.Service
	usageRepository   metering.UsageRepository

	googleClientID     string
	googleClientSecret string
}

func NewProviderService(
	imapServerService imapserver.Service,
	usageRepository metering.UsageRepository,
	googleClientID,
	googleClientSecret string,
) *DefaultProviderService {
	return &DefaultProviderService{
		imapServerService,
		usageRepository,
		googleClientID,
		googleClientSecret,
	}
}

// Gmail quota is 2500mb/day, allow IMAP clients other than us to keep working
var defaultQuota int = metering.Megabyte * 1800

const domainGmail = "gmail.com"

func (s DefaultProviderService) GetProvider(ctx context.Context, address string) (Provider, error) {
	domain, err := mailboxsweep.DomainFromEmail(address)
	if err != nil {
		return nil, err
	}

	var provider Provider
	switch domain {
	case domainGmail:
		provider = NewGmailProvider(ctx, s.googleClientID, s.googleClientSecret)

	// default to IMAP
	default:
		server, err := s.imapServerService.Find(ctx)
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
