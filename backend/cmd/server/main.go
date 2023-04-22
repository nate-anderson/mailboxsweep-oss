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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
	stripeClient "github.com/stripe/stripe-go/v74/client"
)

type serverConfig struct {
	HTTPPort     int    `fig:"HTTP_PORT"`
	Env          string `fig:"ENV"`
	LoginBaseURL string `fig:"BASE_URL"`

	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION"`

	S3Host   *string `fig:"S3_HOST"`
	S3Port   int     `fig:"S3_PORT"`
	S3Region string  `fig:"S3_REGION"`

	SQSHost     *string `fig:"SQS_HOST"`
	SQSPort     int     `fig:"SQS_PORT"`
	SQSRegion   string  `fig:"SQS_REGION"`
	SQSQueueURL string  `fig:"SQS_QUEUE_URL"`

	DynamoDBTable             string `fig:"DYNAMODB_TABLE"`
	S3Bucket                  string `fig:"S3_BUCKET"`
	CredentialCipherKeyBase64 string `fig:"CREDENTIAL_CIPHER_KEY_BASE64"`

	LoginLinkDurationMinutes int    `fig:"LOGIN_LINK_DURATION_MINUTES"`
	LoginTokenLength         int    `fig:"LOGIN_TOKEN_LENGTH"`
	SessionKeyLength         int    `fig:"SESSION_KEY_LENGTH"`
	SessionDurationMinutes   int    `fig:"SESSION_DURATION_MINUTES"`
	CookieDomain             string `fig:"COOKIE_DOMAIN"`
	CookiePath               string `fig:"COOKIE_PATH"`

	// How long must a user wait before submitting a new classification request, when the previous one hasn't finished?
	// 3 hours?
	MinutesBetweenOpenRequests int `fig:"MINUTES_BETWEEN_OPEN_REQUESTS"`
	// How long must a user wait after a completed request to submit a new one?
	HoursBetweenClosedRequests int `fig:"MINUTES_BETWEEN_CLOSED_REQUESTS"`

	// Used to split changesets up among documents
	ChangeSetPageIDHashMod int64 `fig:"CHANGESET_PAGE_ID_HASH_MOD"`

	DevMode bool `fig:"DEV_MODE"`

	// how long should inbox metadata (folders and message counts) be cached?
	InboxMetadataTTLMinutes int `fig:"INBOX_METADATA_TTL_MINUTES"`

	// How long is a payment good for access to the servuce?
	PaymentDurationDays int `fig:"PAYMENT_ACCESS_DURATION_DAYS"`

	StripeAPIKey                         string `fig:"STRIPE_API_KEY"`
	StripeCheckoutSessionDurationMinutes int    `fig:"STRIPE_CHECKOUT_SESSION_DURATION_MINUTES"`
	StripePaymentConfirmationURL         string `fig:"STRIPE_PAYMENT_CONFIRMATION_URL"`

	CORSAllowedOrigin string `fig:"CORS_ALLOWED_ORIGIN"`

	SigningKeyFilePath string `fig:"SIGNING_KEY_FILE_PATH"`

	ClassificationPageSize int `fig:"CLASSIFICATION_PAGE_SIZE"`

	GoogleClientID        string `fig:"GOOGLE_OAUTH_CLIENT_ID"`
	GoogleClientSecret    string `fig:"GOOGLE_OAUTH_CLIENT_SECRET"`
	GoogleAuthRedirectURL string `fig:"GOOGLE_OAUTH_REDIRECT_URL"`
}

func main() {
	log := logrus.New()

	envDriver, err := fig.NewEnvironmentDriver(".env")
	if err != nil {
		log.Fatal(err)
	}

	conf := fig.New(envDriver)

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

	signingKeyFile, err := os.Open(cfg.SigningKeyFilePath)
	if err != nil {
		log.Fatalf("failed to open signing key file %s: %s", cfg.SigningKeyFilePath, err)
	}
	cert, err := login.ReadCert(signingKeyFile)
	if err != nil {
		log.Fatalf("failed to read signing cert: %s", err)
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
				return
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
