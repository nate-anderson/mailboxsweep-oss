package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type workerConfig struct {
	DynamoDBHost   *string `mapstructure:"DYNAMODB_HOST"`
	DynamoDBPort   int     `mapstructure:"DYNAMODB_PORT"`
	DynamoDBRegion string  `mapstructure:"DYNAMODB_REGION"`

	S3Host   *string `mapstructure:"S3_HOST"`
	S3Port   int     `mapstructure:"S3_PORT"`
	S3Region string  `mapstructure:"S3_REGION"`

	SQSHost   *string `mapstructure:"SQS_HOST"`
	SQSPort   int     `mapstructure:"SQS_PORT"`
	SQSRegion string  `mapstructure:"SQS_REGION"`

	S3Bucket      string `mapstructure:"S3_BUCKET"`
	EventBusName  string `mapstructure:"EVENT_BUS_NAME"`
	DynamoDBTable string `mapstructure:"DYNAMODB_TABLE"`
	SQSQueueURL   string `mapstructure:"SQS_QUEUE_URL"`

	CredentialCipherKeyBase64 string `mapstructure:"CREDENTIAL_CIPHER_KEY_BASE64"`

	// How long must a user wait before submitting a new classification request, when the previous one hasn't finished?
	// 3 hours?
	MinutesBetweenOpenRequests int `mapstructure:"MINUTES_BETWEEN_OPEN_REQUESTS"`
	// How long must a user wait after a completed request to submit a new one?
	HoursBetweenClosedRequests int `mapstructure:"MINUTES_BETWEEN_CLOSED_REQUESTS"`
}

func main() {
	log := logrus.New()

	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	cfg := workerConfig{}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	db, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.DynamoDBHost,
			ServicePort:   cfg.DynamoDBPort,
			ServiceRegion: cfg.DynamoDBRegion,
		},
		dynamodb.NewFromConfig,
	)
	if err != nil {
		log.Fatal(err)
	}

	credentialRepo := inboxcredential.NewDynamoInboxCredentialRepository(db, cfg.DynamoDBTable)

	cipherKey, err := base64.RawStdEncoding.DecodeString(cfg.CredentialCipherKeyBase64)
	if err != nil {
		log.Fatalf("failed to decode configured AES Cipher key: %s", err)
	}

	credentialService := inboxcredential.NewService(credentialRepo, cipherKey)

	imapServerRepo := imapserver.NewConstantIMAPServerRepository()
	imapServerService := imapserver.NewService(imapServerRepo)
	providerService := inbox.NewProviderService(imapServerService)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, cfg.DynamoDBTable)
	userService := inboxuser.NewService(userRepo)

	timeBetweenOpenRequests := time.Minute * time.Duration(cfg.MinutesBetweenOpenRequests)
	timeBetweenClosedRequests := time.Hour * time.Duration(cfg.HoursBetweenClosedRequests)
	classifyRequestRepo := classifyrequest.NewDynamoRequestRepository(db, cfg.DynamoDBTable)
	classifyRequestService := classifyrequest.NewDefaultService(userService, classifyRequestRepo, timeBetweenOpenRequests, timeBetweenClosedRequests)

	s3, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.S3Host,
			ServicePort:   cfg.S3Port,
			ServiceRegion: cfg.S3Region,
		},
		s3.NewFromConfig,
		func(o *s3.Options) {
			o.UsePathStyle = true
		},
	)
	if err != nil {
		log.Fatalf("failed to instantiate S3 client: %s", err)
	}

	classificationRepo := classification.NewS3ClassificationRepository(s3, cfg.S3Bucket)

	queue, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.SQSHost,
			ServicePort:   cfg.SQSPort,
			ServiceRegion: cfg.SQSRegion,
		},
		sqs.NewFromConfig,
	)
	if err != nil {
		log.Fatalf("failed to initialize eventbridge client: %s", err)
	}

	pub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queue, cfg.SQSQueueURL)

	service := classification.NewService(
		credentialService,
		imapServerService,
		classificationRepo,
		providerService,
		classifyRequestService,
		log,
		pub,
		cfg.EventBusName,
	)

	if err != nil {
		log.Fatalf("failed to initialize SQS client: %s", err)
	}

	for {

		ctx := context.Background()
		retrieveID, err := uuid.NewUUID()
		if err != nil {
			log.Errorf("failed to generate UUID to read from queue: %s", err)
			continue
		}
		batch, err := queue.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:                &cfg.SQSQueueURL,
			ReceiveRequestAttemptId: aws.String(retrieveID.String()),
		})
		if err != nil {
			log.
				WithError(err).
				Errorf("failed to read message from queue %s", cfg.SQSQueueURL)
			continue
		}

		/*
			@TODO improvement
			This is a simple loop queue reader
			This should use a Pool to allow concurrent queue worker
			goroutines
		*/

		for _, m := range batch.Messages {
			queueItem := classification.ClassifyQueueItem{}
			err := json.Unmarshal([]byte(*m.Body), &queueItem)
			if err != nil {
				log.Errorf("failed to unmarshal queue item: %s", err)
				continue
			}

			classifiers := make([]classification.Classifier, len(queueItem.Classifiers))
			for i, srcl := range queueItem.Classifiers {
				cl, err := srcl.Classifier()
				if err != nil {
					log.Errorf("invalid classifier %v: %s", srcl, err)
					continue
				}
				classifiers[i] = cl
			}

			service.Classify(ctx, queueItem.Address, queueItem.Folders, classifiers)
		}
	}
}
