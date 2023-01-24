package imapserver

import "context"

type Server struct {
	// the domain in the email
	Domain string `dynamodbav:"domain"`

	// the IMAP host and port
	Host string `dynamodbav:"host"`
	Port int    `dynamodbav:"port"`
}

type ServerRepository interface {
	FindByDomain(ctx context.Context, domain string) (*Server, error)
	Store(ctx context.Context, server Server) error
}
