package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxtesting"
	"mailboxsweep/inboxuser"
	"mailboxsweep/payment"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Integration test

// publishing to SQS queue results in worker doing work

// worker failing work results in same data on DLQ

type testImapServerService struct {
	t              *testing.T
	expectedDomain string
	wasCalled      bool
}

func (s *testImapServerService) FindByDomain(ctx context.Context, domain string) (*imapserver.Server, error) {
	assert.That(s.t, domain).Equals(domain)
	s.wasCalled = true
	return &imapserver.Server{
		Domain: s.expectedDomain,
		Host:   "imap." + s.expectedDomain,
		Port:   993,
	}, nil
}

func (s testImapServerService) Store(ctx context.Context, server imapserver.Server) error {
	s.t.Fatalf("don't call Store on testImapServerService")
	return nil
}

func TestWorker(t *testing.T) {
	t.Skip() // @TODO - figure out how to get this test working
	ctx := context.Background()
	log := logrus.New()

	client, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   aws.String("localhost"),
			ServicePort:   4566,
			ServiceRegion: "us-west-2",
		},
		sqs.NewFromConfig,
	)
	assert.ThatError(t, err).IsNil()

	createRes, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("worker_testqueue.fifo"),
		Attributes: map[string]string{
			"FifoQueue": "true",
		},
	})
	assert.ThatError(t, err).IsNil()

	queueURL := createRes.QueueUrl

	createDQLRes, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("worker_test_dlq.fifo"),
		Attributes: map[string]string{
			"FifoQueue": "true",
		},
	})
	assert.ThatError(t, err).IsNil()

	dlqURL := createDQLRes.QueueUrl

	defer func() {
		_, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: queueURL,
		})
		if err != nil {
			t.Fatal("failed to tear down test queue: %w", err)
		}

		_, err = client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: dlqURL,
		})
		if err != nil {
			t.Fatal("failed to tear down test DLQ: %w", err)
		}
	}()

	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	dlqPub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](client, *dlqURL)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	userService, err := inboxuser.NewService(userRepo)

	// store user
	testAddress := "classifyuser@example.org"
	testIMAPPassword := "test_password"
	folders := []string{"INBOX", "trash"}

	_, err = userService.Store(ctx, testAddress)
	assert.ThatError(t, err).IsNil()

	testProcessor := payment.NewTestProcessor()
	testProcessor.SessionID = "12345"
	cartRepo := payment.NewCartRepository(db, table)
	paymentService, err := payment.NewService(userService, testProcessor, cartRepo, time.Hour)
	assert.ThatError(t, err).IsNil()

	credentialReadService := &inboxtesting.TestCredentialReadService{
		Address:  testAddress,
		Password: testIMAPPassword,
	}

	// simulate payment
	authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, testAddress)
	_, err = paymentService.InitiatePayment(authedCtx, inboxuser.Basic)
	assert.ThatError(t, err).IsNil()

	err = paymentService.CompletePayment(authedCtx, inboxuser.PaymentMethod("Test"), testProcessor.SessionID)
	assert.ThatError(t, err).IsNil()

	imapServerService := &testImapServerService{
		t:              t,
		expectedDomain: "example.org",
	}
	classificationRepo := classification.NewFileClassificationRepository(t.TempDir())
	providerService := inboxtesting.NewTestProviderService()
	numMessages := 100
	userNames := []string{
		"me",
		"you",
		"them",
	}
	inboxMessages := make([]inbox.Message, numMessages)
	for i := 0; i < numMessages; i++ {
		un := userNames[i%3]
		sender := fmt.Sprintf("%s@gmail.com", un)
		msg := inbox.Message{
			UID:     uint32(i),
			Folder:  "INBOX",
			Subject: fmt.Sprintf("msg %d", i),
			Sender: inbox.Sender{
				Address:  sender,
				Name:     un,
				Username: un,
				Domain:   "gmail.com",
			},
			SendDate: time.Now().Add(time.Hour * 24 * 365 * -1 * time.Duration(i)),
		}
		inboxMessages[i] = msg
	}
	providerService.Messages["INBOX"] = []inbox.Message{}
	providerService.Messages["trash"] = []inbox.Message{}

	assert.ThatError(t, err).IsNil()
	classifyRequestRepo := classifyrequest.NewDynamoRequestRepository(db, table)
	classifyRequestService := classifyrequest.NewDefaultService(userService, classifyRequestRepo, time.Hour, time.Hour)

	// seed an open request
	err = classifyRequestService.RecordRequest(authedCtx)
	assert.ThatError(t, err).IsNil()

	eventPublisher := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](client, *queueURL)
	classificationService := classification.NewService(
		credentialReadService,
		imapServerService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		log,
		eventPublisher,
	)

	wrk := NewWorker(log, client, *queueURL, dlqPub, *dlqURL, classificationService)

	go func() {
		err = wrk.Work()
		assert.ThatError(t, err).IsNil()
	}()

	t.Run("classification requests enqueued are picked up by worker", func(t *testing.T) {
		err = eventPublisher.PublishEvent(ctx, mailboxsweep.EventBody[classification.ClassifyQueueItem]{
			RequestID: aws.String("test-request"),
			Body: classification.ClassifyQueueItem{
				Address: testAddress,
				Folders: folders,
				Classifiers: []classification.SerializableClassifier{
					{
						TypeKey: classification.TypeKeySenderDomain,
					},
					{
						TypeKey: classification.TypeKeySenderUsername,
					},
				},
			},
		})
		assert.ThatError(t, err).IsNil()

		time.Sleep(time.Second * 3)

		// the imapserverservice has been queried for the correct host
		assert.That(t, imapServerService.wasCalled).Equals(true)

		// the credential service has been queried for the correct address
		assert.That(t, credentialReadService.FindCredentialCalled).Equals(1)

		// the classification service has a summary for the requested user
		root, err := classificationService.ClassificationSummary(authedCtx)
		assert.ThatError(t, err).IsNil()
		assert.That(t, root.Classification.Count).Equals(uint(numMessages))
	})

	t.Run("failed requests populate dlq", func(t *testing.T) {
		// provoke failure
		credentialReadService.FindError = errors.New("test error")

		err = eventPublisher.PublishEvent(ctx, mailboxsweep.EventBody[classification.ClassifyQueueItem]{
			RequestID: aws.String("test-request"),
			Body: classification.ClassifyQueueItem{
				Address: testAddress,
				Folders: folders,
				Classifiers: []classification.SerializableClassifier{
					{
						TypeKey: classification.TypeKeySenderDomain,
					},
					{
						TypeKey: classification.TypeKeySenderUsername,
					},
				},
			},
		})
		assert.ThatError(t, err).IsNil()

		time.Sleep(time.Second * 3)

		retrieveID, err := uuid.NewUUID()
		assert.ThatError(t, err).IsNil()

		batch, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:                dlqURL,
			ReceiveRequestAttemptId: aws.String(retrieveID.String()),
			WaitTimeSeconds:         30,
		})
		assert.ThatError(t, err).IsNil()
		assert.That(t, len(batch.Messages)).Equals(1)

		queueItem := mailboxsweep.EventBody[classification.ClassifyQueueItem]{}
		err = json.Unmarshal([]byte(*batch.Messages[0].Body), &queueItem)
		assert.ThatError(t, err).IsNil()

		assert.That(t, queueItem.Body.Address).Equals(testAddress)
		assert.ThatSlice(t, queueItem.Body.Classifiers).HasLength(2)
		assert.ThatSlice(t, queueItem.Body.Folders).ContainsAll(folders)
	})
}
