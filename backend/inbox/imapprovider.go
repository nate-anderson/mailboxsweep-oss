package inbox

import (
	"fmt"
	"mailboxsweep"
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

func NewIMAPProvider(host string, port int, quotaBytes *int) (*IMAPProvider, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	opts := []metering.DialerOpt{}
	if quotaBytes != nil {
		opts = append(opts, metering.WithQuota(*quotaBytes))
	}
	dialer := metering.NewDialer()
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

func (p *IMAPProvider) Connect() error {
	addr := fmt.Sprintf("%s:%d", p.IMAPHost, p.IMAPPort)
	cl, err := client.DialTLS(addr, nil)
	if err != nil {
		return err
	}

	p.client = cl
	return nil
}

func (p *IMAPProvider) Login(username, password string) (Inbox, error) {
	err := p.client.Login(username, password)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid credentials") {
			return nil, mailboxsweep.BadInboxCredentialError(err)
		}
		return nil, err
	}

	return NewIMAPInbox(p.client, username), nil
}
