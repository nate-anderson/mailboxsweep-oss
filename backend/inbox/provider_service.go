package inbox

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/imapserver"
	"mailboxsweep/metering"

	"github.com/sirupsen/logrus"
)

type ProviderService interface {
	GetProvider(ctx context.Context, address string) (Provider, error)
}

type DefaultProviderService struct {
	imapServerService imapserver.Service
	usageRepository   metering.UsageRepository

	googleClientID     string
	googleClientSecret string

	logger *logrus.Logger

	gmailAPILimits GmailAPIRateLimits
}

func NewProviderService(
	imapServerService imapserver.Service,
	usageRepository metering.UsageRepository,
	googleClientID,
	googleClientSecret string,
	gmailAPIRateLimits GmailAPIRateLimits,
	logger *logrus.Logger,
) *DefaultProviderService {
	return &DefaultProviderService{
		imapServerService,
		usageRepository,
		googleClientID,
		googleClientSecret,
		logger,
		gmailAPIRateLimits,
	}
}

// Gmail quota is 2500mb/day, allow IMAP clients other than us to keep working
var defaultQuota int = metering.Megabyte * 1800

func (s DefaultProviderService) GetProvider(ctx context.Context, address string) (Provider, error) {
	domain, err := mailboxsweep.DomainFromEmail(address)
	if err != nil {
		return nil, err
	}

	var provider Provider
	switch domain {
	case mailboxsweep.DomainGmail:
		provider = NewGmailProvider(ctx, s.googleClientID, s.googleClientSecret, s.gmailAPILimits, s.logger)

	// default to IMAP
	default:
		server, err := s.imapServerService.Find(ctx)
		if err != nil {
			return nil, err
		}

		provider, err = NewIMAPProvider(ctx, server.Host, server.Port, address, s.usageRepository, &defaultQuota, s.logger)
		if err != nil {
			return nil, err
		}
	}

	return provider, nil
}
