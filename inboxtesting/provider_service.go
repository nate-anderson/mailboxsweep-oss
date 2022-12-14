package inboxtesting

import (
	"context"
	"mailboxsweep/inbox"
)

type TestProviderService struct {
	GetProviderError error
}

func NewTestProviderService() *TestProviderService {
	return &TestProviderService{}
}

func (t TestProviderService) GetProvider(ctx context.Context, address string) (inbox.Provider, error) {
	if t.GetProviderError != nil {
		return nil, t.GetProviderError
	}
	return &TestProvider{}, nil
}
