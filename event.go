package mailboxsweep

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventPublisher interface {
	PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type TestEventPublisher struct {
	Entries []types.PutEventsRequestEntry

	PutEventsError error
}

func (t *TestEventPublisher) PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	if t.PutEventsError != nil {
		return nil, t.PutEventsError
	}

	t.Entries = append(t.Entries, params.Entries...)
	return &eventbridge.PutEventsOutput{}, nil
}
