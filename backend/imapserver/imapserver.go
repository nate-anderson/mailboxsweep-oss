package imapserver

import "context"

type Server struct {
	// the domain in the email
	Domain string `dynamodbav:"domain"`
	// is this available to all users, or just the one who created it?
	IsGlobal  bool    `dynamodbav:"is_global"`
	CreatedBy *string `dynamodbav:"created_by"`

	// the IMAP host and port
	Host string `dynamodbav:"host"`
	Port int    `dynamodbav:"port"`
}

type ServerRepository interface {
	FindByDomain(ctx context.Context, domain string) (*Server, error)
	FindByUserAddress(ctx context.Context, address string) (*Server, error)
	Store(ctx context.Context, server Server) error
}
