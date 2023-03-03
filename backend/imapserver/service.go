package imapserver

import (
	"context"
	"errors"
	"mailboxsweep"
	"strings"
)

type Service interface {
	Find(ctx context.Context) (*Server, error)
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

func (s DefaultService) Find(ctx context.Context) (*Server, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domain, err := mailboxsweep.DomainFromEmail(addr)
	if err != nil {
		return nil, err
	}

	// attempt to find global server first
	server, err := s.repo.FindByDomain(ctx, domain)
	if errors.Is(err, mailboxsweep.ErrNotFound) || (server != nil && !server.IsGlobal) {
		return s.repo.FindByUserAddress(ctx, addr)
	} else if err != nil {
		return nil, err
	}

	return server, nil
}

var (
	ErrInvalidDomain = errors.New("invalid domain")
	ErrInvalidHost   = errors.New("invalid host")
	ErrInvalidPort   = errors.New("invalid port")
)

func (s DefaultService) Store(ctx context.Context, server Server) error {
	userAddr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

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
		Host:      cleanHost,
		Domain:    cleanDomain,
		Port:      server.Port,
		IsGlobal:  false,
		CreatedBy: &userAddr,
	}

	_, err = mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.Store(ctx, cleanServer)
}

func (s DefaultService) FindByUserAddress(ctx context.Context) (*Server, error) {
	userAddress, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByUserAddress(ctx, userAddress)
}
