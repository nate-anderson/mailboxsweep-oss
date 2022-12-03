package imapserver

import "context"

type Service interface {
	FindByDomain(ctx context.Context, domain string) (*Server, error)
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
	return s.repo.FindByDomain(ctx, domain)
}
