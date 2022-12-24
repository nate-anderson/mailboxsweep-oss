package worker

import (
	"context"
	"encoding/json"
	"mailboxsweep"
	"mailboxsweep/classification"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// worker watches a queue and classifies inboxes
type Worker struct {
	log      *logrus.Logger
	queue    *sqs.Client
	queueURL string

	dlqPub  mailboxsweep.EventPublisher[classification.ClassifyQueueItem]
	dlqURL  string
	service classification.Service
}

func NewWorker(
	log *logrus.Logger,
	queue *sqs.Client,
	queueURL string,
	dlqPub mailboxsweep.EventPublisher[classification.ClassifyQueueItem],
	dlqURL string,
	service classification.Service,
) *Worker {
	return &Worker{
		log,
		queue,
		queueURL,
		dlqPub,
		dlqURL,
		service,
	}
}

func (w Worker) Work() error {
	w.log.Infof("Listening to queue %s...\n", w.queueURL)

	for {
		ctx := context.Background()
		retrieveID, err := uuid.NewUUID()
		if err != nil {
			w.log.Errorf("failed to generate UUID to read from queue: %s", err)
			continue
		}

		batch, err := w.queue.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:                &w.queueURL,
			ReceiveRequestAttemptId: aws.String(retrieveID.String()),
			WaitTimeSeconds:         30,
		})
		if err != nil {
			w.log.
				WithError(err).
				Errorf("failed to read message from queue %s", w.queueURL)
			continue
		}

		/*
			@TODO improvement
			This is a simple loop queue reader
			This should use a Pool to allow concurrent queue worker
			goroutines
		*/

		w.log.Infof("read %d messages from the queue", len(batch.Messages))

		for _, m := range batch.Messages {
			queueItem := mailboxsweep.EventBody[classification.ClassifyQueueItem]{}
			err := json.Unmarshal([]byte(*m.Body), &queueItem)
			if err != nil {
				w.log.Errorf("failed to unmarshal queue item: %s", err)
				continue
			}

			body := queueItem.Body
			if strings.TrimSpace(body.Address) == "" {
				w.log.Errorf("no address in queue item %+v", body)
			}
			jobCtx := mailboxsweep.AuthorizeJobContext(ctx, body.Address)

			classifiers := make([]classification.Classifier, len(body.Classifiers))
			for i, srcl := range body.Classifiers {
				cl, err := srcl.Classifier()
				if err != nil {
					w.log.Errorf("invalid classifier %v: %s", srcl, err)
					continue
				}
				classifiers[i] = cl
			}

			err = w.service.Classify(jobCtx, body.Folders, classifiers)
			if err != nil {
				w.log.
					WithError(err).
					Error("classify failed")
				// push to DLQ
				err = w.dlqPub.PublishEvent(ctx, queueItem)
				if err != nil {
					w.log.
						WithError(err).
						Error("failed to push to DLQ after failure")
				} else {
					_, err = w.queue.DeleteMessage(jobCtx, &sqs.DeleteMessageInput{
						QueueUrl:      &w.queueURL,
						ReceiptHandle: m.ReceiptHandle,
					})
					if err != nil {
						w.log.Errorf("failed to delete message after failed classify + DLQ enqueue: %s", err.Error())
					}
				}
			} else {
				w.log.Info("classify complete")
				_, err = w.queue.DeleteMessage(jobCtx, &sqs.DeleteMessageInput{
					QueueUrl:      &w.queueURL,
					ReceiptHandle: m.ReceiptHandle,
				})
				if err != nil {
					w.log.Errorf("failed to delete message after classify: %s", err.Error())
				}
			}
		}
	}
}
