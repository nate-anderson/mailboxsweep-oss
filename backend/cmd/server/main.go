package main

import (
	"crypto/rsa"
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
	"mailboxsweep/metering"
	"mailboxsweep/payment"
	"mailboxsweep/session"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
	stripeClient "github.com/stripe/stripe-go/v74/client"
)

type serverConfig struct {
	HTTPPort     int    `fig:"HTTP_PORT" required:"true"`
	Env          string `fig:"ENV" default:"prod"`
	LoginBaseURL string `fig:"BASE_URL" default:"https://localhost:3001/login-token"`

	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION" default:"us-west-2"`

	S3Host   *string `fig:"S3_HOST"`
	S3Port   int     `fig:"S3_PORT"`
	S3Region string  `fig:"S3_REGION" default:"us-west-2"`

	SQSHost     *string `fig:"SQS_HOST"`
	SQSPort     int     `fig:"SQS_PORT"`
	SQSRegion   string  `fig:"SQS_REGION" default:"us-west-2"`
	SQSQueueURL string  `fig:"SQS_QUEUE_URL" required:"true"`

	DynamoDBTable             string `fig:"DYNAMODB_TABLE" required:"true"`
	S3Bucket                  string `fig:"S3_BUCKET" required:"true"`
	CredentialCipherKeyBase64 string `fig:"CREDENTIAL_CIPHER_KEY_BASE64" required:"true"`

	LoginLinkDurationMinutes int    `fig:"LOGIN_LINK_DURATION_MINUTES" default:"10"`
	LoginTokenLength         int    `fig:"LOGIN_TOKEN_LENGTH" default:"30"`
	SessionKeyLength         int    `fig:"SESSION_KEY_LENGTH" default:"30"`
	SessionDurationMinutes   int    `fig:"SESSION_DURATION_MINUTES" default:"120"`
	CookieDomain             string `fig:"COOKIE_DOMAIN" required:"true"`
	CookiePath               string `fig:"COOKIE_PATH" required:"true"`

	// How long must a user wait before submitting a new classification request, when the previous one hasn't finished?
	// 3 hours?
	MinutesBetweenOpenRequests int `fig:"MINUTES_BETWEEN_OPEN_REQUESTS" required:"true"`
	// How long must a user wait after a completed request to submit a new one?
	HoursBetweenClosedRequests int `fig:"MINUTES_BETWEEN_CLOSED_REQUESTS" required:"true"`

	// Used to split changesets up among documents
	ChangeSetPageIDHashMod int64 `fig:"CHANGESET_PAGE_ID_HASH_MOD" default:"11"`

	DevMode bool `fig:"DEV_MODE" default:"false"`

	// how long should inbox metadata (folders and message counts) be cached?
	InboxMetadataTTLMinutes int `fig:"INBOX_METADATA_TTL_MINUTES" default:"120"`

	// How long is a payment good for access to the servuce?
	PaymentDurationDays int `fig:"PAYMENT_ACCESS_DURATION_DAYS" required:"true"`

	StripeAPIKey                         string `fig:"STRIPE_API_KEY" required:"true"`
	StripeCheckoutSessionDurationMinutes int    `fig:"STRIPE_CHECKOUT_SESSION_DURATION_MINUTES" required:"true"`
	StripePaymentConfirmationURL         string `fig:"STRIPE_PAYMENT_CONFIRMATION_URL" required:"true"`

	CORSAllowedOrigin string `fig:"CORS_ALLOWED_ORIGIN" required:"true"`

	SigningKeyFilePath string  `fig:"SIGNING_KEY_FILE_PATH"`
	SigningKey         *string `fig:"SIGNING_KEY"`

	ClassificationPageSize int `fig:"CLASSIFICATION_PAGE_SIZE" default:"30"`

	GoogleClientID        string `fig:"GOOGLE_OAUTH_CLIENT_ID" required:"true"`
	GoogleClientSecret    string `fig:"GOOGLE_OAUTH_CLIENT_SECRET" required:"true"`
	GoogleAuthRedirectURL string `fig:"GOOGLE_OAUTH_REDIRECT_URL" required:"true"`
}

func main() {
	log := logrus.New()

	envDriver, err := fig.NewEnvironmentDriver(".env")
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

	// default to env then paramstore
	conf := fig.New(envDriver, ssmDriver)

	cfg := serverConfig{}
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

	cipherKey, err := base64.RawStdEncoding.DecodeString(cfg.CredentialCipherKeyBase64)
	if err != nil {
		log.Fatalf("failed to decode configured AES Cipher key: %s", err)
	}

	credentialRepo := inboxcredential.NewDynamoInboxCredentialRepository(db, cfg.DynamoDBTable)
	credentialService, err := inboxcredential.NewService(credentialRepo, cipherKey)
	if err != nil {
		log.Fatal(err)
	}

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

	usageRepo := metering.NewUsageRepository(db, cfg.DynamoDBTable)

	providerService := inbox.NewProviderService(
		imapServerService,
		usageRepo,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
	)

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

	metadataRepo := inbox.NewMetadataRepository(db, cfg.DynamoDBTable)
	if cfg.InboxMetadataTTLMinutes <= 0 {
		err = errors.New("InboxMetadataTTLMinutes must be greater than zero")
		log.Fatal(err)
	}
	metadataService := inbox.NewMetadataService(metadataRepo, providerService, credentialService, time.Duration(cfg.InboxMetadataTTLMinutes)*time.Minute)

	classificationRepo := classification.NewS3ClassificationRepository(s3, cfg.S3Bucket, cfg.ClassificationPageSize)
	classificationService := classification.NewService(
		credentialService,
		imapServerService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		metadataService,
		log,
		publisher,
	)

	loginRepo := login.NewDynamoLinkRepository(db, cfg.DynamoDBTable)

	sessionRepo := session.NewDynamoSessionRepository(db, cfg.DynamoDBTable)
	if cfg.SessionDurationMinutes == 0 {
		log.Fatal("session duration is required")
	}
	sessionDuration := time.Minute * time.Duration(cfg.SessionDurationMinutes)
	sessionService := session.NewService(sessionRepo, cfg.SessionKeyLength, sessionDuration)

	var loginSender login.Sender

	if cfg.Env == "dev" {
		loginSender = login.NewLogSender(log, cfg.LoginBaseURL)
	}

	if cfg.LoginLinkDurationMinutes == 0 {
		log.Fatal("LOGIN_LINK_DURATION_MINUTES must be set and > 0")
	}
	if cfg.LoginTokenLength == 0 {
		log.Fatal("LOGIN_TOKEN_LENGTH must be set and > 0")
	}
	loginLinkDurationMinutes := time.Minute * time.Duration(cfg.LoginLinkDurationMinutes)

	var cert *rsa.PrivateKey
	if cfg.SigningKey != nil {
		cert, err = login.ReadCert(strings.NewReader(*cfg.SigningKey))
		if err != nil {
			log.Fatalf("failed to instantiate signing key from configured data; %s", err)
		}
	} else {
		log.Printf("No signing key configured, looking for a key at '%s'\n", cfg.SigningKeyFilePath)
		signingKeyFile, err := os.Open(cfg.SigningKeyFilePath)
		if err != nil {
			log.Fatalf("failed to open signing key file %s: %s", cfg.SigningKeyFilePath, err)
		}
		cert, err = login.ReadCert(signingKeyFile)
		if err != nil {
			log.Fatalf("failed to read signing cert: %s", err)
		}
	}

	gmailTokenRepository := login.NewDynamoGmailTokenRepository(db, cfg.DynamoDBTable)

	loginLinkService := login.NewLinkService(loginRepo, userService, loginSender, cert, loginLinkDurationMinutes, cfg.LoginTokenLength)
	gmailLoginService := login.NewGmailLoginService(gmailTokenRepository, credentialService, userService, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleAuthRedirectURL)
	loginService := login.NewService(loginLinkService, gmailLoginService, userService, sessionService, log)

	if strings.TrimSpace(cfg.CookieDomain) == "" || strings.TrimSpace(cfg.CookiePath) == "" {
		log.Fatal("cookie domain and path are not configured")
	}

	changesetRepo, err := changeset.NewDynamoChangesetRepository(db, cfg.DynamoDBTable, uint64(cfg.ChangeSetPageIDHashMod))
	if err != nil {
		log.Fatal(err)
	}

	changesetService := changeset.NewDefaultService(changesetRepo)
	if cfg.StripeAPIKey == "" {
		log.Fatal("stripe API key must be set")
	}

	inboxChangesetService := inboxchangeset.NewService(
		changesetService,
		classificationService,
		providerService,
		credentialService,
		classifyRequestService,
		metadataService,
	)

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
		imapServerService,
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
			}),
		).ServeHTTP,
	)

	loggingFormatter := middleware.DefaultLogFormatter{
		Logger:  log,
		NoColor: !cfg.DevMode,
	}
	loggingMiddleware := middleware.RequestLogger(&loggingFormatter)
	srv := httpserver.HandlerWithOptions(controller, httpserver.ChiServerOptions{
		BaseRouter: root,
		Middlewares: []httpserver.MiddlewareFunc{
			auth.Middleware,
			cors.Middleware,
			loggingMiddleware,
		},
	})

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)

	log.Infof("Starting Mailboxsweep HTTP server at %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
