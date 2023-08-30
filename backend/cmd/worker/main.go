package main

import (
	"context"
	"encoding/base64"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/email"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxchangeset"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/metering"
	"mailboxsweep/payment"
	"mailboxsweep/worker"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
	stripeClient "github.com/stripe/stripe-go/v74/client"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type workerConfig struct {
	Env            string  `fig:"ENV" default:"prod"`
	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION" default:"us-west-2"`

	S3Host   *string `fig:"S3_HOST"`
	S3Port   int     `fig:"S3_PORT"`
	S3Region string  `fig:"S3_REGION" default:"us-west-2"`

	SQSHost   *string `fig:"SQS_HOST"`
	SQSPort   int     `fig:"SQS_PORT"`
	SQSRegion string  `fig:"SQS_REGION" default:"us-west-2"`

	SESHost   *string `fig:"SES_HOST"`
	SESPort   int     `fig:"SES_PORT"`
	SESRegion string  `fig:"SES_REGION" default:"us-west-2"`

	S3Bucket                      string `fig:"S3_BUCKET" required:"true"`
	DynamoDBTable                 string `fig:"DYNAMODB_TABLE" required:"true"`
	SQSClassifyQueueURL           string `fig:"SQS_CLASSIFY_QUEUE_URL" required:"true"`
	SQSCommitQueueURL             string `fig:"SQS_COMMIT_QUEUE_URL" required:"true"`
	SQSClassifyDeadLetterQueueURL string `fig:"SQS_CLASSIFY_DLQ_URL" required:"true"`
	SQSCommitDeadLetterQueueURL   string `fig:"SQS_COMMIT_DLQ_URL" required:"true"`

	CredentialCipherKeyBase64 string `fig:"CREDENTIAL_CIPHER_KEY_BASE64"`

	// How long must a user wait before submitting a new classification request, when the previous one hasn't finished?
	// 3 hours?
	MinutesBetweenOpenRequests int `fig:"MINUTES_BETWEEN_OPEN_REQUESTS"`
	// How long must a user wait after a completed request to submit a new one?
	MinutesBetweenClosedRequests int `fig:"MINUTES_BETWEEN_CLOSED_REQUESTS"`

	// How long is a payment good for access to the servuce?
	PaymentDurationDays int `fig:"PAYMENT_ACCESS_DURATION_DAYS"`

	StripeAPIKey                         string `fig:"STRIPE_API_KEY" required:"true"`
	StripeCheckoutSessionDurationMinutes int    `fig:"STRIPE_CHECKOUT_SESSION_DURATION_MINUTES" required:"true"`
	StripePaymentConfirmationURL         string `fig:"STRIPE_PAYMENT_CONFIRMATION_URL" required:"true"`

	ClassificationPageSize        int `fig:"CLASSIFICATION_PAGE_SIZE" default:"30"`
	ClassificationSubjectPreviews int `fig:"CLASSIFICATION_SUBJECT_PREVIEWS" default:"5"`

	InboxMetadataTTLMinutes int `fig:"INBOX_METADATA_TTL_MINUTES" default:"120"`

	GoogleClientID     string `fig:"GOOGLE_OAUTH_CLIENT_ID" required:"true"`
	GoogleClientSecret string `fig:"GOOGLE_OAUTH_CLIENT_SECRET" required:"true"`

	GmailDeleteCallCost       int     `fig:"GMAIL_API_DELETE_CALL_COST"        default:"50"`
	GmailMarkReadCallCost     int     `fig:"GMAIL_API_MARK_READ_CALL_COST"     default:"50"`
	GmailListMessagesCallCost int     `fig:"GMAIL_API_LIST_MESSAGES_CALL_COST" default:"5"`
	GmailGetMessageCallCost   int     `fig:"GMAIL_API_GET_MESSAGE_CALL_COST"   default:"5"`
	GmailAPICallLimit         float64 `fig:"GMAIL_API_CALL_LIMIT"              default:"250"`

	EmailReplyToAddress string `fig:"EMAIL_REPLY_TO_ADDRESS" default:"no-reply@mailboxsweep.com"`
	EmailSenderName     string `fig:"EMAIL_SENDER_NAME" default:"MailboxSweep"`

	SESClassifyCompleteEmailTemplateName string `fig:"SES_CLASSIFY_COMPLETE_EMAIL_TEMPLATE_NAME" default:"mailboxsweep_classify_complete"`

	ClassifySelectURL string `fig:"CLASSIFY_SELECT_URL"`
	SESConfigSetName  string `fig:"SES_CONFIGURATION_SET_NAME" default:"MailboxSweep_Transactional"`

	// Used to split changesets up among documents
	ChangeSetPageIDHashMod int64 `fig:"CHANGESET_PAGE_ID_HASH_MOD" default:"11"`
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

	if cfg.Env == "staging" || cfg.Env == "prod" {
		log.Formatter = &logrus.JSONFormatter{
			PrettyPrint:     false,
			TimestampFormat: time.RFC3339,
		}
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

	gmailRateLimits := inbox.GmailAPIRateLimits{
		APICallLimit:         rate.Limit(cfg.GmailAPICallLimit),
		DeleteCallCost:       cfg.GmailDeleteCallCost,
		MarkReadCallCost:     cfg.GmailMarkReadCallCost,
		ListMessagesCallCost: cfg.GmailListMessagesCallCost,
		GetMessageCallCost:   cfg.GmailGetMessageCallCost,
	}
	providerService := inbox.NewProviderService(imapServerService, usageRepo, cfg.GoogleClientID, cfg.GoogleClientSecret, gmailRateLimits, log)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, cfg.DynamoDBTable)
	if err != nil {
		log.Fatalf("failed to initialize user repo: %s", err)
	}
	userService, err := inboxuser.NewService(userRepo)
	if err != nil {
		log.Fatal(err)
	}

	timeBetweenOpenRequests := time.Minute * time.Duration(cfg.MinutesBetweenOpenRequests)
	timeBetweenClosedRequests := time.Minute * time.Duration(cfg.MinutesBetweenClosedRequests)
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

	queueClient, err := mailboxsweep.ConnectAWSService(
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

	pub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queueClient, cfg.SQSClassifyQueueURL)

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

	classificationService := classification.NewService(
		credentialService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		metadataService,
		userService,
		log,
		pub,
		cfg.ClassificationSubjectPreviews,
	)

	classifyDLQPub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queueClient, cfg.SQSClassifyDeadLetterQueueURL)
	commitDLQPub := mailboxsweep.NewSQSEventPublisher[changeset.CommitQueueMessage](queueClient, cfg.SQSCommitDeadLetterQueueURL)

	var emailSender email.Sender
	if cfg.Env == "staging" || cfg.Env == "prod" {
		log.WithField("environment", cfg.Env).Info("using SES email sender")
		sesClient, err := mailboxsweep.ConnectAWSService(mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.SQSHost,
			ServicePort:   cfg.SQSPort,
			ServiceRegion: cfg.SQSRegion,
			Environment:   cfg.Env,
		}, sesv2.NewFromConfig)
		if err != nil {
			log.Fatal(err)
			return
		}

		emailSender = email.NewSESSender(sesClient, cfg.EmailReplyToAddress, cfg.EmailSenderName, cfg.SESConfigSetName)
	}

	changesetRepo, err := changeset.NewDynamoChangesetRepository(db, cfg.DynamoDBTable, uint64(cfg.ChangeSetPageIDHashMod))
	if err != nil {
		log.Fatal(err)
	}

	changesetService := changeset.NewDefaultService(changesetRepo)
	inboxChangsesetService := inboxchangeset.NewService(
		changesetService,
		classificationService,
		providerService,
		credentialService,
		classifyRequestService,
		metadataService,
		commitDLQPub,
		log,
	)

	classifyWorker := worker.NewClassifyWorker(
		log,
		queueClient,
		cfg.SQSClassifyQueueURL,
		classifyDLQPub,
		cfg.SQSClassifyDeadLetterQueueURL,
		classificationService,
		emailSender,
		cfg.SESClassifyCompleteEmailTemplateName,
		userService,
		cfg.ClassifySelectURL,
	)

	commitWorker := worker.NewCommitWorker(
		log,
		inboxChangsesetService,
		queueClient,
		cfg.SQSCommitQueueURL,
		commitDLQPub,
		cfg.SQSCommitDeadLetterQueueURL,
	)

	group, workerCtx := errgroup.WithContext(context.Background())
	group.Go(func() error {
		return classifyWorker.Work(workerCtx)
	})
	group.Go(func() error {
		return commitWorker.Work(workerCtx)
	})
	err = group.Wait()
	log.Fatal(err)
}
