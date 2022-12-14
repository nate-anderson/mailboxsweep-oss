package inboxtesting

import (
	"context"
	"mailboxsweep/inbox"
)

type TestProviderService struct {
	GetProviderError error

	Messages map[string][]inbox.Message
}

func NewTestProviderService() *TestProviderService {
	return &TestProviderService{
		Messages: make(map[string][]inbox.Message),
	}
}

func (t TestProviderService) GetProvider(ctx context.Context, address string) (inbox.Provider, error) {
	if t.GetProviderError != nil {
		return nil, t.GetProviderError
	}
	return &TestProvider{
		Messages: t.Messages,
	}, nil
}
