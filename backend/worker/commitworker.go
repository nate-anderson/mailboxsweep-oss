package worker

import (
	"context"
	"encoding/json"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// subset of inboxchangeset service
type ChangeCommitter interface {
	CommitChanges(ctx context.Context, operation changeset.UpdateOperation) error
}

// CommitWorker watches a queue and commits changesets (mark read or delete)
type CommitWorker struct {
	log      *logrus.Logger
	queue    *sqs.Client
	queueURL string

	service ChangeCommitter

	dlqPub mailboxsweep.EventPublisher[changeset.CommitQueueMessage]
	dlqURL string
}

func NewCommitWorker(
	log *logrus.Logger,
	service ChangeCommitter,
	queue *sqs.Client,
	queueURL string,
	dlqPub mailboxsweep.EventPublisher[changeset.CommitQueueMessage],
	dlqURL string,
) *CommitWorker {
	return &CommitWorker{
		log:      log,
		queue:    queue,
		service:  service,
		queueURL: queueURL,
		dlqPub:   dlqPub,
		dlqURL:   dlqURL,
	}
}

const commitVisibilityTimeoutMinutes = 1

func (w CommitWorker) Work(ctx context.Context) error {
	log := w.log.WithField("worker", "commit")
	log.Infof("Listening to commit queue %s...\n", w.queueURL)

	for {
		retrieveID, err := uuid.NewUUID()
		if err != nil {
			log.Errorf("failed to generate UUID to read from queue: %s", err)
			continue
		}

		batch, err := w.queue.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:                &w.queueURL,
			ReceiveRequestAttemptId: aws.String(retrieveID.String()),
			WaitTimeSeconds:         20,
			VisibilityTimeout:       60 * (commitVisibilityTimeoutMinutes + 1),
		})
		if err != nil {
			log.
				WithError(err).
				Errorf("failed to read message from queue %s", w.queueURL)
			continue
		}

		log.Infof("read %d messages from the queue", len(batch.Messages))
		wg := sync.WaitGroup{}
		wg.Add(len(batch.Messages))
		for _, m := range batch.Messages {
			go func(message types.Message) {
				defer wg.Done()
				var e mailboxsweep.EventBody[changeset.CommitQueueMessage]
				if err := json.Unmarshal([]byte(*message.Body), &e); err != nil {
					log.Errorf("failed to unmarshal queue item: %s", err)
					return
				}

				jobCtx := mailboxsweep.AuthorizeJobContext(ctx, e.Body.Inbox)
				log := log.
					WithField("address", e.Body.Inbox).
					WithField("operation", e.Body.Operation)

				if err = w.service.CommitChanges(jobCtx, e.Body.Operation); err != nil {
					log.Error(err)

					if err = w.dlqPub.PublishEvent(jobCtx, e); err != nil {
						log.
							WithError(err).
							Error("failed to push to DLQ after failure")

						log = log.WithField("dlq publish error", err)
					}
				}

				if _, err = w.queue.DeleteMessage(jobCtx, &sqs.DeleteMessageInput{
					QueueUrl:      &w.queueURL,
					ReceiptHandle: message.ReceiptHandle,
				}); err != nil {
					log.
						WithField("queue delete error", err).
						Errorf("failed to delete message from queue")
				}
			}(m)

		}
	}
}
