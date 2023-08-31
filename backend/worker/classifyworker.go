package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/email"
	"mailboxsweep/inboxuser"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ClassifyWorker watches a queue and classifies inboxes
type ClassifyWorker struct {
	log      *logrus.Logger
	queue    *sqs.Client
	queueURL string

	dlqPub  mailboxsweep.EventPublisher[classification.ClassifyQueueItem]
	dlqURL  string
	service classification.Service

	emailSender              email.Sender
	classifyCompleteTemplate string
	inboxUserService         inboxuser.Service

	changesetSelectURL string
}

func NewClassifyWorker(
	log *logrus.Logger,
	queue *sqs.Client,
	queueURL string,
	dlqPub mailboxsweep.EventPublisher[classification.ClassifyQueueItem],
	dlqURL string,
	service classification.Service,
	emailSender email.Sender,
	classifyCompleteTemplate string,
	inboxUserService inboxuser.Service,
	changesetSelectURL string,
) *ClassifyWorker {
	return &ClassifyWorker{
		log,
		queue,
		queueURL,
		dlqPub,
		dlqURL,
		service,
		emailSender,
		classifyCompleteTemplate,
		inboxUserService,
		changesetSelectURL,
	}
}

const classifyVisibilityTimeoutMinutes = 5

func (w ClassifyWorker) Work(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in ClassifyWorker.Work: %v", r)
		}
	}()

	log := w.log.WithField("worker", "classify")
	log.Infof("Listening to classify queue %s...\n", w.queueURL)

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
			VisibilityTimeout:       60 * (classifyVisibilityTimeoutMinutes + 1),
			MaxNumberOfMessages:     1,
		})
		if err != nil {
			log.
				WithError(err).
				Errorf("failed to read message from queue %s", w.queueURL)
			continue
		}

		/*
			@TODO improvement
			This is a simple loop queue reader
			This should use a Pool to allow concurrent queue worker
			goroutines
			Increase ReceiveMessage call MaxNumberOfMessages
		*/

		log.Infof("read %d messages from the queue", len(batch.Messages))

		wg := sync.WaitGroup{}
		wg.Add(len(batch.Messages))
		for _, m := range batch.Messages {
			start := time.Now()
			queueItem := mailboxsweep.EventBody[classification.ClassifyQueueItem]{}
			err := json.Unmarshal([]byte(*m.Body), &queueItem)
			if err != nil {
				log.Errorf("failed to unmarshal queue item: %s", err)
				continue
			}

			body := queueItem.Body
			if strings.TrimSpace(body.Address) == "" {
				log.Errorf("no address in queue item %+v", body)
				continue
			}
			jobCtx := mailboxsweep.AuthorizeJobContext(ctx, body.Address)
			log := log.
				WithField("address", body.Address).
				WithField("num_classifiers", len(body.Classifiers))

			log.Info("starting inbox classification")

			classifiers := make([]classification.Classifier, len(body.Classifiers))
			for i, srcl := range body.Classifiers {
				cl, err := srcl.Classifier()
				if err != nil {
					log.Errorf("invalid classifier %v: %s", srcl, err)
					continue
				}
				classifiers[i] = cl
			}

			errChan := make(chan error)
			// keep the message handle active while the process runs
			go func(address string, m types.Message) {
				ticker := time.NewTicker(time.Minute * 1)
				failedAttempts := 0
				defer ticker.Stop()
				defer wg.Done()
				for {
					select {
					case <-ticker.C:
						// mark message as being processed
						_, err = w.queue.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
							QueueUrl:          &w.queueURL,
							ReceiptHandle:     m.ReceiptHandle,
							VisibilityTimeout: 60 * classifyVisibilityTimeoutMinutes,
						})
						if err != nil {
							log.
								WithError(err).
								Errorf("failed to ChangeMessageVisibility for message %s (receipt handle %s)", *m.MessageId, *m.ReceiptHandle)
							failedAttempts++
							if failedAttempts >= 3 {
								log.WithError(err).Error("failed 3 attempts to change message visibility, aborting")
								return
							}

						}
					case err := <-errChan:
						if err != nil {
							log.Errorf("got error for message, releasing handle", *m.MessageId, *m.ReceiptHandle)
							_, err = w.queue.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
								QueueUrl:          &w.queueURL,
								ReceiptHandle:     m.ReceiptHandle,
								VisibilityTimeout: 0,
							})
							if err != nil {
								log.Errorf("failed to release handle %s after error: %w", *m.ReceiptHandle, err)
							}
						}
						return
					}
				}
			}(body.Address, m)

			end := time.Now()
			err = w.service.Classify(jobCtx, body.Folders, classifiers)
			log.
				WithError(err).
				Info("CALL TO CLASSIFY RETURNED")

			if err != nil {
				errChan <- err
				log.
					WithError(err).
					WithField("execution_time_seconds", end.Sub(start).Seconds()).
					Error("classify failed")
					// push to DLQ
					/*
								err = w.dlqPub.PublishEvent(ctx, queueItem)
								if err != nil {
									log.
										WithError(err).
										Error("failed to push to DLQ after failure")
								} else {

						_, err = w.queue.DeleteMessage(jobCtx, &sqs.DeleteMessageInput{
							QueueUrl:      &w.queueURL,
							ReceiptHandle: m.ReceiptHandle,
						})
						if err != nil {
							log.Errorf("failed to delete message after failed classify + DLQ enqueue: %s", err.Error())
						}
						}
					*/
				continue
			} else {
				errChan <- nil
			}

			log.Info("going to delete message after classify")
			_, err = w.queue.DeleteMessage(jobCtx, &sqs.DeleteMessageInput{
				QueueUrl:      &w.queueURL,
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				log.
					WithError(err).
					Errorf("failed to delete message after classify")
			}

			log.
				WithField("execution_time_seconds", end.Sub(start).Seconds()).
				Info("classify complete")

			var greeting string
			user, err := w.inboxUserService.FindByAddress(jobCtx, body.Address)
			// this user must exist, this failure can only be temporary
			// default to the generic greeting
			if err != nil {
				log.
					WithError(err).
					Error("failed to find user after completing classification - continuing with generic email greeting")

				greeting = "Hi there,"
			} else if user.NameFirst != "" {
				greeting = fmt.Sprintf("Hi, %s,", user.NameFirst)
			}

			if w.emailSender != nil {
				log.Infof("going to send completion email (template %s, recipient %s)", w.classifyCompleteTemplate, body.Address)
				if err := w.emailSender.Send(ctx, body.Address, w.classifyCompleteTemplate, map[string]string{
					"greeting":       greeting,
					"changeset_link": w.changesetSelectURL,
				}); err != nil {
					log.
						WithField("address", body.Address).
						WithError(err).
						Error("failed to send notification email for classification completion")
				} else {
					log.Info("sent notification email")
				}
			} else {
				log.Infof("skipping email notification - no sender configured (w.emailSender == %+v)", w.emailSender)
			}
		}
		wg.Wait()
	}
}
