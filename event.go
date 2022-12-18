package mailboxsweep

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type EventBody[t any] struct {
	RequestID string `json:"-"`
	Body      t      `json:"body"`
}
type EventPublisher[t any] interface {
	PublishEvent(ctx context.Context, body EventBody[t]) error
}

type TestEventPublisher[t any] struct {
	Entries []EventBody[t]

	PutEventsError error
}

func (p *TestEventPublisher[t]) PublishEvent(ctx context.Context, body EventBody[t]) error {
	if p.PutEventsError != nil {
		return p.PutEventsError
	}

	p.Entries = append(p.Entries, body)
	return nil
}

type SQSEventPublisher[t any] struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSEventPublisher[t any](client *sqs.Client, queueURL string) *SQSEventPublisher[t] {
	return &SQSEventPublisher[t]{
		client:   client,
		queueURL: queueURL,
	}
}

func (p SQSEventPublisher[t]) PublishEvent(ctx context.Context, body EventBody[t]) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(bodyBytes)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"request_id": {
				DataType:    aws.String("String"),
				StringValue: &body.RequestID,
			},
		},
		QueueUrl: &p.queueURL,
	})
	return err
}
