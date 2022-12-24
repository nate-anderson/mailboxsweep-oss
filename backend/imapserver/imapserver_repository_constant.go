package imapserver

import (
	"context"
	"fmt"
)

type ConstantIMAPServerRepository struct {
	db map[string]Server
}

func NewConstantIMAPServerRepository() *ConstantIMAPServerRepository {
	return &ConstantIMAPServerRepository{
		db: map[string]Server{
			"gmail.com": {
				Host: "imap.gmail.com",
				Port: 993,
			},
		},
	}
}

func (r ConstantIMAPServerRepository) FindByDomain(ctx context.Context, domain string) (*Server, error) {
	srv, ok := r.db[domain]
	if !ok {
		return nil, fmt.Errorf("server with domain %s not found", domain)
	}
	return &srv, nil
}
