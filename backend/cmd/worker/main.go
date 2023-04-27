package main

import (
	"encoding/base64"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/metering"
	"mailboxsweep/payment"
	"mailboxsweep/worker"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
	stripeClient "github.com/stripe/stripe-go/v74/client"
)

type workerConfig struct {
	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION" default:"us-west-2"`

	S3Host   *string `fig:"S3_HOST"`
	S3Port   int     `fig:"S3_PORT"`
	S3Region string  `fig:"S3_REGION" default:"us-west-2"`

	SQSHost   *string `fig:"SQS_HOST"`
	SQSPort   int     `fig:"SQS_PORT"`
	SQSRegion string  `fig:"SQS_REGION" default:"us-west-2"`

	S3Bucket              string `fig:"S3_BUCKET" required:"true"`
	DynamoDBTable         string `fig:"DYNAMODB_TABLE" required:"true"`
	SQSQueueURL           string `fig:"SQS_QUEUE_URL" required:"true"`
	SQSDeadLetterQueueURL string `fig:"SQS_DLQ_URL" required:"true"`

	CredentialCipherKeyBase64 string `fig:"CREDENTIAL_CIPHER_KEY_BASE64"`

	// How long must a user wait before submitting a new classification request, when the previous one hasn't finished?
	// 3 hours?
	MinutesBetweenOpenRequests int `fig:"MINUTES_BETWEEN_OPEN_REQUESTS"`
	// How long must a user wait after a completed request to submit a new one?
	HoursBetweenClosedRequests int `fig:"MINUTES_BETWEEN_CLOSED_REQUESTS"`

	// How long is a payment good for access to the servuce?
	PaymentDurationDays int `fig:"PAYMENT_ACCESS_DURATION_DAYS"`

	StripeAPIKey                         string `fig:"STRIPE_API_KEY" required:"true"`
	StripeCheckoutSessionDurationMinutes int    `fig:"STRIPE_CHECKOUT_SESSION_DURATION_MINUTES" required:"true"`
	StripePaymentConfirmationURL         string `fig:"STRIPE_PAYMENT_CONFIRMATION_URL" required:"true"`

	ClassificationPageSize int `fig:"CLASSIFICATION_PAGE_SIZE" default:"30"`

	InboxMetadataTTLMinutes int `fig:"INBOX_METADATA_TTL_MINUTES" default:"120"`

	GoogleClientID     string `fig:"GOOGLE_OAUTH_CLIENT_ID" required:"true"`
	GoogleClientSecret string `fig:"GOOGLE_OAUTH_CLIENT_SECRET" required:"true"`
}

func main() {
	log := logrus.New()

	envDriver, err := fig.NewOptionalFileEnvironmentDriver(".env")
	if err != nil {
		log.Fatal(err)
	}

	paramstoreClient, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceRegion: "us-west-2",
		},
		ssm.NewFromConfig,
	)
	if err != nil {
		log.Fatal(err)
	}

	ssmDriver := mailboxsweep.NewSSMConfigDriver(paramstoreClient)

	conf := fig.New(envDriver, ssmDriver)

	cfg := workerConfig{}
	if err := conf.Unmarshal(&cfg); err != nil {
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

	credentialService, err := inboxcredential.NewService(credentialRepo, cipherKey)
	if err != nil {
		log.Fatal(err)
	}

	imapServerRepo := imapserver.NewIMAPServerRepository(db, cfg.DynamoDBTable)
	imapServerService := imapserver.NewService(imapServerRepo)

	usageRepo := metering.NewUsageRepository(db, cfg.DynamoDBTable)

	providerService := inbox.NewProviderService(imapServerService, usageRepo, cfg.GoogleClientID, cfg.GoogleClientSecret)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, cfg.DynamoDBTable)
	if err != nil {
		log.Fatalf("failed to initialize user repo: %s", err)
	}
	userService, err := inboxuser.NewService(userRepo)
	if err != nil {
		log.Fatal(err)
	}

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

	classificationRepo := classification.NewS3ClassificationRepository(s3, cfg.S3Bucket, cfg.ClassificationPageSize)

	queue, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.SQSHost,
			ServicePort:   cfg.SQSPort,
			ServiceRegion: cfg.SQSRegion,
		},
		sqs.NewFromConfig,
	)
	if err != nil {
		log.Fatalf("failed to initialize SQS client: %s", err)
	}

	pub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queue, cfg.SQSQueueURL)

	stripeClient := stripeClient.New(cfg.StripeAPIKey, nil)
	checkoutSessionDuration := time.Minute * time.Duration(cfg.StripeCheckoutSessionDurationMinutes)
	stripeProcessor, err := payment.NewStripeProcessor(stripeClient, cfg.StripePaymentConfirmationURL, checkoutSessionDuration)
	if err != nil {
		log.Fatal(err)
	}
	day := time.Hour * 24
	paymentAccessDuration := time.Duration(cfg.PaymentDurationDays) * day
	cartRepo := payment.NewCartRepository(db, cfg.DynamoDBTable)
	paymentService, err := payment.NewService(userService, stripeProcessor, cartRepo, paymentAccessDuration)
	if err != nil {
		log.Fatal(err)
	}

	metadataRepo := inbox.NewMetadataRepository(db, cfg.DynamoDBTable)
	metadataService := inbox.NewMetadataService(
		metadataRepo,
		providerService,
		credentialService,
		time.Duration(cfg.InboxMetadataTTLMinutes)*time.Minute,
	)

	service := classification.NewService(
		credentialService,
		imapServerService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		metadataService,
		log,
		pub,
	)

	dlqPub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queue, cfg.SQSDeadLetterQueueURL)

	w := worker.NewWorker(
		log,
		queue,
		cfg.SQSQueueURL,
		dlqPub,
		cfg.SQSDeadLetterQueueURL,
		service,
	)

	log.Fatal(w.Work())
}
