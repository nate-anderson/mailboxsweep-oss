package inboxtesting

import (
	"context"
	"mailboxsweep/inbox"
)

type TestProviderService struct {
	GetProviderError error

	Messages map[string][]inbox.Message

	// maintain reference for testing
	// @TODO: figure out a better way to mock than this
	// one provider per address, using pointers so we can retrieve and
	// run assertions against the folder
	Providers map[string]*TestProvider
}

func NewTestProviderService() *TestProviderService {
	return &TestProviderService{
		Messages:  make(map[string][]inbox.Message),
		Providers: make(map[string]*TestProvider),
	}
}

func (t TestProviderService) GetProvider(ctx context.Context, address string) (inbox.Provider, error) {
	if t.GetProviderError != nil {
		return nil, t.GetProviderError
	}

	existing, ok := t.Providers[address]
	if ok {
		return existing, nil
	}

	pr := &TestProvider{
		Messages: t.Messages,
		Inboxes:  make(map[string]*TestInbox),
	}
	t.Providers[address] = pr
	return pr, nil
}
