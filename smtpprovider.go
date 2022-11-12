package mailsweeper

import (
	"fmt"
	"net"
	"strconv"

	"github.com/emersion/go-imap/client"
)

type SMTPProvider struct {
	SMTPHost string
	SMTPPort int

	client *client.Client
}

func NewSMTPProvider(host string, port int) (*SMTPProvider, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	cl, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}

	return &SMTPProvider{
		SMTPHost: host,
		SMTPPort: port,
		client:   cl,
	}, nil
}

func (p *SMTPProvider) Connect() error {
	addr := fmt.Sprintf("%s:%d", p.SMTPHost, p.SMTPPort)
	cl, err := client.DialTLS(addr, nil)
	if err != nil {
		return err
	}

	p.client = cl
	return nil
}

func (p *SMTPProvider) Login(username, password string) (Inbox, error) {
	err := p.client.Login(username, password)
	if err != nil {
		return nil, err
	}

	return NewSMTPInbox(p.client), nil
}
