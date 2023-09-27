package inbox

import (
	"context"
	"crypto/tls"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/metering"
	"net"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/client"
	"github.com/sirupsen/logrus"
)

type IMAPProvider struct {
	IMAPHost string
	IMAPPort int

	client *client.Client
	logger *logrus.Logger
}

func NewIMAPProvider(
	ctx context.Context,
	host string,
	port int,
	address string,
	usageRepo metering.UsageRepository,
	quotaBytes *int,
	logger *logrus.Logger,
	requireTLS bool,
) (*IMAPProvider, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	opts := []metering.DialerOpt{}
	if quotaBytes != nil {
		opts = append(opts, metering.WithQuota(*quotaBytes))
	}
	dialer := metering.NewDialer(ctx, address, usageRepo, opts...)
	var config *tls.Config
	if !requireTLS {
		logger.Warnf("IMAP provider configured not to require TLS")
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	cl, err := client.DialWithDialerTLS(dialer, addr, config)
	if err != nil {
		return nil, err
	}

	return &IMAPProvider{
		IMAPHost: host,
		IMAPPort: port,
		client:   cl,
		logger:   logger,
	}, nil
}

func (p *IMAPProvider) Connect(_ context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.IMAPHost, p.IMAPPort)
	cl, err := client.DialTLS(addr, nil)
	if err != nil {
		return err
	}

	p.client = cl
	return nil
}

func (p *IMAPProvider) Login(_ context.Context, username string, credential inboxcredential.Credential) (Inbox, error) {
	imapCred, ok := credential.(*inboxcredential.IMAPInboxCredential)
	if !ok {
		return nil, fmt.Errorf("credential type mismatch: expected IMAP, got %t", credential)
	}

	err := p.client.Login(username, imapCred.Password)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid credentials") {
			return nil, mailboxsweep.BadInboxCredentialError(err)
		}
		return nil, err
	}

	return NewIMAPInbox(p.client, username, p.logger), nil
}

func (p *IMAPProvider) WrapError(err error) error {
	return mailboxsweep.IMAPError(err)
}
