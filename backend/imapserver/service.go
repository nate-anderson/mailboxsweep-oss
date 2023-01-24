package imapserver

import (
	"context"
	"errors"
	"mailboxsweep"
	"strings"
)

type Service interface {
	FindByDomain(ctx context.Context, domain string) (*Server, error)
	Store(ctx context.Context, server Server) error
}

type DefaultService struct {
	repo ServerRepository
}

func NewService(repo ServerRepository) Service {
	return &DefaultService{
		repo: repo,
	}
}

func (s DefaultService) FindByDomain(ctx context.Context, domain string) (*Server, error) {
	_, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByDomain(ctx, domain)
}

var (
	ErrInvalidDomain = errors.New("invalid domain")
	ErrInvalidHost   = errors.New("invalid host")
	ErrInvalidPort   = errors.New("invalid port")
)

func (s DefaultService) Store(ctx context.Context, server Server) error {
	cleanDomain := strings.TrimSpace(server.Domain)
	if cleanDomain == "" || !strings.Contains(cleanDomain, ".") {
		return ErrInvalidDomain
	}

	cleanHost := strings.TrimSpace(server.Host)
	if cleanHost == "" || !strings.Contains(cleanHost, ".") {
		return ErrInvalidHost
	}

	if server.Port == 0 {
		return ErrInvalidPort
	}

	cleanServer := Server{
		Host:   cleanHost,
		Domain: cleanDomain,
		Port:   server.Port,
	}

	_, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.Store(ctx, cleanServer)
}
