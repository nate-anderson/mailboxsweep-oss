package inbox

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/metering"
	"net"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/client"
)

type IMAPProvider struct {
	IMAPHost string
	IMAPPort int

	client *client.Client
}

func NewIMAPProvider(
	ctx context.Context,
	host string,
	port int,
	address string,
	usageRepo metering.UsageRepository,
	quotaBytes *int,
) (*IMAPProvider, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	opts := []metering.DialerOpt{}
	if quotaBytes != nil {
		opts = append(opts, metering.WithQuota(*quotaBytes))
	}
	dialer := metering.NewDialer(ctx, address, usageRepo, opts...)
	cl, err := client.DialWithDialerTLS(dialer, addr, nil)
	if err != nil {
		return nil, err
	}

	return &IMAPProvider{
		IMAPHost: host,
		IMAPPort: port,
		client:   cl,
	}, nil
}

func NewTestIMAPProvider(host string, port int) (*IMAPProvider, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	cl, err := client.Dial(addr)
	if err != nil {
		return nil, err
	}

	return &IMAPProvider{
		IMAPHost: host,
		IMAPPort: port,
		client:   cl,
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

	return NewIMAPInbox(p.client, username), nil
}
