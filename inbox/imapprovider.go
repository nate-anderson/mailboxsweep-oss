package inbox

import (
	"fmt"
	"net"
	"strconv"

	"github.com/emersion/go-imap/client"
)

type IMAPProvider struct {
	IMAPHost string
	IMAPPort int

	client *client.Client
}

func NewIMAPProvider(host string, port int) (*IMAPProvider, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	cl, err := client.DialTLS(addr, nil)
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
		return nil, err
	}

	return NewIMAPInbox(p.client, username), nil
}
