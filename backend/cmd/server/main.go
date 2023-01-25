package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/httpserver"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxchangeset"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/login"
	"mailboxsweep/payment"
	"mailboxsweep/session"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	stripeClient "github.com/stripe/stripe-go/v74/client"
)

type serverConfig struct {
	HTTPPort int    `mapstructure:"HTTP_PORT"`
	Env      string `mapstructure:"ENV"`
	SiteURL  string `mapstructure:"BASE_URL"`

	DynamoDBHost   *string `mapstructure:"DYNAMODB_HOST"`
	DynamoDBPort   int     `mapstructure:"DYNAMODB_PORT"`
	DynamoDBRegion string  `mapstructure:"DYNAMODB_REGION"`

	S3Host   *string `mapstructure:"S3_HOST"`
	S3Port   int     `mapstructure:"S3_PORT"`
	S3Region string  `mapstructure:"S3_REGION"`

	SQSHost     *string `mapstructure:"SQS_HOST"`
	SQSPort     int     `mapstructure:"SQS_PORT"`
	SQSRegion   string  `mapstructure:"SQS_REGION"`
	SQSQueueURL string  `mapstructure:"SQS_QUEUE_URL"`

	DynamoDBTable             string `mapstructure:"DYNAMODB_TABLE"`
	S3Bucket                  string `mapstructure:"S3_BUCKET"`
	EventBusName              string `mapstructure:"EVENT_BUS_NAME"`
	CredentialCipherKeyBase64 string `mapstructure:"CREDENTIAL_CIPHER_KEY_BASE64"`

	LoginLinkDurationMinutes int    `mapstructure:"LOGIN_LINK_DURATION_MINUTES"`
	LoginTokenLength         int    `mapstructure:"LOGIN_TOKEN_LENGTH"`
	SessionKeyLength         int    `mapstructure:"SESSION_KEY_LENGTH"`
	SessionDurationMinutes   int    `mapstructure:"SESSION_DURATION_MINUTES"`
	CookieDomain             string `mapstructure:"COOKIE_DOMAIN"`
	CookiePath               string `mapstructure:"COOKIE_PATH"`

	// How long must a user wait before submitting a new classification request, when the previous one hasn't finished?
	// 3 hours?
	MinutesBetweenOpenRequests int `mapstructure:"MINUTES_BETWEEN_OPEN_REQUESTS"`
	// How long must a user wait after a completed request to submit a new one?
	HoursBetweenClosedRequests int `mapstructure:"MINUTES_BETWEEN_CLOSED_REQUESTS"`

	// Used to split changesets up among documents
	ChangeSetPageIDHashMod uint64 `mapstructure:"CHANGESET_PAGE_ID_HASH_MOD"`

	DevMode bool `mapstructure:"DEV_MODE"`

	// how long should inbox metadata (folders and message counts) be cached?
	InboxMetadataTTLMinutes int `mapstructure:"INBOX_METADATA_TTL_MINUTES"`

	// How long is a payment good for access to the servuce?
	PaymentDurationDays int `mapstructure:"PAYMENT_ACCESS_DURATION_DAYS"`

	StripeAPIKey                         string `mapstructure:"STRIPE_API_KEY"`
	StripeCheckoutSessionDurationMinutes int    `mapstructure:"STRIPE_CHECKOUT_SESSION_DURATION_MINUTES"`
	StripePaymentConfirmationURL         string `mapstructure:"STRIPE_PAYMENT_CONFIRMATION_URL"`

	CORSAllowedOrigin string `mapstructure:"CORS_ALLOWED_ORIGIN"`
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

	cfg := serverConfig{}
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

	cipherKey, err := base64.RawStdEncoding.DecodeString(cfg.CredentialCipherKeyBase64)
	if err != nil {
		log.Fatalf("failed to decode configured AES Cipher key: %s", err)
	}

	credentialRepo := inboxcredential.NewDynamoInboxCredentialRepository(db, cfg.DynamoDBTable)
	credentialService := inboxcredential.NewService(credentialRepo, cipherKey)

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

	imapServerRepo := imapserver.NewIMAPServerRepository(db, cfg.DynamoDBTable)
	imapServerService := imapserver.NewService(imapServerRepo)
	providerService := inbox.NewProviderService(imapServerService)

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

	publisher := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queue, cfg.SQSQueueURL)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, cfg.DynamoDBTable)
	userService, err := inboxuser.NewService(userRepo)
	if err != nil {
		log.Fatalf("failed to initialize inboxuser service: %s", err)
	}

	timeBetweenOpenRequests := time.Minute * time.Duration(cfg.MinutesBetweenOpenRequests)
	timeBetweenClosedRequests := time.Hour * time.Duration(cfg.HoursBetweenClosedRequests)
	classifyRequestRepo := classifyrequest.NewDynamoRequestRepository(db, cfg.DynamoDBTable)
	classifyRequestService := classifyrequest.NewDefaultService(userService, classifyRequestRepo, timeBetweenOpenRequests, timeBetweenClosedRequests)

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

	classificationRepo := classification.NewS3ClassificationRepository(s3, cfg.S3Bucket)
	classificationService := classification.NewService(
		credentialService,
		imapServerService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		log,
		publisher,
		cfg.EventBusName,
	)

	metadataRepo := inbox.NewMetadataRepository(db, cfg.DynamoDBTable)
	if cfg.InboxMetadataTTLMinutes <= 0 {
		err = errors.New("InboxMetadataTTLMinutes must be greater than zero")
		log.Fatal(err)
	}
	metadataService := inbox.NewMetadataService(metadataRepo, providerService, credentialService, time.Duration(cfg.InboxMetadataTTLMinutes)*time.Minute)

	loginRepo := login.NewDynamoLinkRepository(db, cfg.DynamoDBTable)

	sessionRepo := session.NewDynamoSessionRepository(db, cfg.DynamoDBTable)
	if cfg.SessionDurationMinutes == 0 {
		log.Fatal("session duration is required")
	}
	sessionDuration := time.Minute * time.Duration(cfg.SessionDurationMinutes)
	sessionService := session.NewService(sessionRepo, cfg.SessionKeyLength, sessionDuration)

	var loginSender login.Sender

	if cfg.Env == "dev" {
		loginSender = login.NewLogSender(log, cfg.SiteURL)
	}

	if cfg.LoginLinkDurationMinutes == 0 {
		log.Fatal("LOGIN_LINK_DURATION_MINUTES must be set and > 0")
	}
	if cfg.LoginTokenLength == 0 {
		log.Fatal("LOGIN_TOKEN_LENGTH must be set and > 0")
	}
	loginLinkDurationMinutes := time.Minute * time.Duration(cfg.LoginLinkDurationMinutes)
	loginService := login.NewService(loginRepo, userService, sessionService, loginSender, loginLinkDurationMinutes, cfg.LoginTokenLength)

	if strings.TrimSpace(cfg.CookieDomain) == "" || strings.TrimSpace(cfg.CookiePath) == "" {
		log.Fatal("cookie domain and path are not configured")
	}

	changesetRepo, err := changeset.NewDynamoChangesetRepository(db, cfg.DynamoDBTable, cfg.ChangeSetPageIDHashMod)
	if err != nil {
		log.Fatal(err)
	}

	changesetService := changeset.NewDefaultService(changesetRepo)
	if cfg.StripeAPIKey == "" {
		log.Fatalf("stripe API key must be set")
	}

	inboxChangesetService := inboxchangeset.NewService(changesetService, classificationService, providerService, credentialService)

	controller := httpserver.NewController(
		userService,
		credentialService,
		classifyRequestService,
		classificationService,
		metadataService,
		providerService,
		loginService,
		changesetService,
		inboxChangesetService,
		paymentService,
		cfg.CookieDomain,
		cfg.CookiePath,
		cfg.DevMode,
		log,
	)

	auth := session.AuthMiddleware{
		Service: sessionService,
	}
	if cfg.CORSAllowedOrigin == "" {
		log.Fatal("CORS_ALLOWED_ORIGIN is not configured")
	}
	cors := httpserver.NewCORSHandler(cfg.CORSAllowedOrigin)

	root := chi.NewMux()
	root.Options("/*",
		cors.Middleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				return
			}),
		).ServeHTTP,
	)

	srv := httpserver.HandlerWithOptions(controller, httpserver.ChiServerOptions{
		BaseRouter: root,
		Middlewares: []httpserver.MiddlewareFunc{
			auth.Middleware,
			cors.Middleware,
		},
	})

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)

	log.Infof("Starting Mailboxsweep HTTP server at %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
