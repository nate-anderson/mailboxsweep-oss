package mailboxsweep

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

type EventBody[t any] struct {
	RequestID *string `json:"-"`
	Body      t       `json:"body"`
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
	msgDedupeID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	input := &sqs.SendMessageInput{
		MessageBody:            aws.String(string(bodyBytes)),
		MessageAttributes:      map[string]types.MessageAttributeValue{},
		MessageGroupId:         &SQSMessageGroup,
		QueueUrl:               &p.queueURL,
		MessageDeduplicationId: aws.String(msgDedupeID.String()),
	}
	if body.RequestID != nil {
		input.MessageAttributes["request_id"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: body.RequestID,
		}
	}
	_, err = p.client.SendMessage(ctx, input)
	return err
}

var SQSMessageGroup = "mailboxsweep_classify"
