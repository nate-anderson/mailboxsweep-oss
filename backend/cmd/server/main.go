package main

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/email"
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
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	xrayECS "github.com/aws/aws-xray-sdk-go/awsplugins/ecs"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
	stripeClient "github.com/stripe/stripe-go/v74/client"
	"golang.org/x/time/rate"
)

type serverConfig struct {
	HTTPPort     int    `fig:"HTTP_PORT" required:"true"`
	Env          string `fig:"ENV" default:"prod"`
	LoginBaseURL string `fig:"BASE_URL" default:"https://localhost:3001/login/token"`

	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION" default:"us-west-2"`

	S3Host   *string `fig:"S3_HOST"`
	S3Port   int     `fig:"S3_PORT"`
	S3Region string  `fig:"S3_REGION" default:"us-west-2"`

	SQSHost   *string `fig:"SQS_HOST"`
	SQSPort   int     `fig:"SQS_PORT"`
	SQSRegion string  `fig:"SQS_REGION" default:"us-west-2"`

	SQSClassifyQueueURL string `fig:"SQS_CLASSIFY_QUEUE_URL" required:"true"`
	SQSCommitQueueURL   string `fig:"SQS_COMMIT_QUEUE_URL" required:"true"`

	SESHost   *string `fig:"SES_HOST"`
	SESPort   int     `fig:"SES_PORT"`
	SESRegion string  `fig:"SES_REGION" default:"us-west-2"`

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
	MinutesBetweenClosedRequests int `fig:"MINUTES_BETWEEN_CLOSED_REQUESTS" required:"true"`

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

	CORSAllowedOriginsConcat string `fig:"CORS_ALLOWED_ORIGINS" required:"true"`

	SigningKeyFilePath string  `fig:"SIGNING_KEY_FILE_PATH"`
	SigningKey         *string `fig:"SIGNING_KEY"`

	ClassificationPageSize int `fig:"CLASSIFICATION_PAGE_SIZE" default:"30"`
	// how many subject previews should be stored in a classification leaf node?
	ClassificationSubjectPreviews int `fig:"CLASSIFICATION_SUBJECT_PREVIEWS" default:"5"`
	// "filesystem" or "s3"
	ClassificationStore               string `fig:"CLASSIFICATION_STORE" default:"s3"`
	ClassificationFilesystemDirectory string `fig:"CLASSIFICATION_FILESYSTEM_DIRECTORY" default:"./classifications"`

	GoogleClientID        string `fig:"GOOGLE_OAUTH_CLIENT_ID" required:"true"`
	GoogleClientSecret    string `fig:"GOOGLE_OAUTH_CLIENT_SECRET" required:"true"`
	GoogleAuthRedirectURL string `fig:"GOOGLE_OAUTH_REDIRECT_URL" required:"true"`

	GmailDeleteCallCost       int     `fig:"GMAIL_API_DELETE_CALL_COST"        default:"50"`
	GmailMarkReadCallCost     int     `fig:"GMAIL_API_MARK_READ_CALL_COST"     default:"50"`
	GmailListMessagesCallCost int     `fig:"GMAIL_API_LIST_MESSAGES_CALL_COST" default:"5"`
	GmailGetMessageCallCost   int     `fig:"GMAIL_API_GET_MESSAGE_CALL_COST"   default:"5"`
	GmailAPICallLimit         float64 `fig:"GMAIL_API_CALL_LIMIT"              default:"250"`

	EmailReplyToAddress string `fig:"EMAIL_REPLY_TO_ADDRESS" default:"no-reply@mailboxsweep.com"`
	EmailSenderName     string `fig:"EMAIL_SENDER_NAME" default:"MailboxSweep"`

	SESLoginEmailTemplateName string `fig:"SES_LOGIN_EMAIL_TEMPLATE_NAME" default:"mailboxsweep_login_link"`
	SESConfigSetName          string `fig:"SES_CONFIGURATION_SET_NAME" default:"MailboxSweep_Transactional"`

	XRayEnabled bool `fig:"XRAY_ENABLED" default:"false"`

	TestLoginServerSenderPort  int     `fig:"TEST_LOGIN_SENDER_SERVER_PORT"`
	TestLoginServerSenderUsers string  `fig:"TEST_LOGIN_SENDER_SERVER_USERS"`
	AllowInsecureIMAP          *bool   `fig:"ALLOW_INSECURE_IMAP" default:"false"`
	UserWhitelistConcat        *string `fig:"USER_WHITELIST"`
}

func main() {
	log := logrus.New()

	// flags:
	// allow specifying configuration file
	var (
		envFile     string
		ssmHostFlag string
		ssmPort     int
	)
	flag.StringVar(&envFile, "envFile", ".env", "specify .env configuration file to load")
	flag.StringVar(&ssmHostFlag, "ssmHost", "", "specify host of SSM (for ParamStore)")
	flag.IntVar(&ssmPort, "ssmPort", 0, "specify port of SSM (for ParamStore)")
	flag.Parse()

	var ssmHost *string
	if ssmHostFlag != "" {
		ssmHost = &ssmHostFlag
	}

	envDriver, err := fig.NewOptionalFileEnvironmentDriver(envFile)
	if err != nil {
		log.Fatal(err)
	}

	paramstoreClient, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceRegion: "us-west-2",
			ServiceHost:   ssmHost,
			ServicePort:   ssmPort,
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

	// use JSON logging when deployed to AWS
	// use ECS XRay config when deployed to AWS
	if (cfg.Env == "prod" || cfg.Env == "staging") && cfg.XRayEnabled {
		xrayECS.Init()
	}

	if !cfg.DevMode {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}

	db, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.DynamoDBHost,
			ServicePort:   cfg.DynamoDBPort,
			ServiceRegion: cfg.DynamoDBRegion,
			Environment:   cfg.Env,
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
			Environment:   cfg.Env,
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

	allowInsecureIMAP := false
	// if insecure IMAP is explicitly configured, and environment is not one we deploy to the internet, allow insecure IMAP
	if cfg.AllowInsecureIMAP != nil && *cfg.AllowInsecureIMAP && cfg.Env != "prod" && cfg.Env != "staging" {
		allowInsecureIMAP = true
	}
	providerService := inbox.NewProviderService(
		imapServerService,
		usageRepo,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		inbox.GmailAPIRateLimits{
			APICallLimit:         rate.Limit(cfg.GmailAPICallLimit),
			DeleteCallCost:       cfg.GmailDeleteCallCost,
			MarkReadCallCost:     cfg.GmailMarkReadCallCost,
			ListMessagesCallCost: cfg.GmailListMessagesCallCost,
			GetMessageCallCost:   cfg.GmailGetMessageCallCost,
		},
		allowInsecureIMAP,
		log,
	)

	queue, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.SQSHost,
			ServicePort:   cfg.SQSPort,
			ServiceRegion: cfg.SQSRegion,
			Environment:   cfg.Env,
		},
		sqs.NewFromConfig,
	)
	if err != nil {
		log.Fatalf("failed to initialize eventbridge client: %s", err)
	}

	classifyQueuePublisher := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](queue, cfg.SQSClassifyQueueURL)
	commitQueuePublisher := mailboxsweep.NewSQSEventPublisher[changeset.CommitQueueMessage](queue, cfg.SQSCommitQueueURL)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, cfg.DynamoDBTable)
	userService, err := inboxuser.NewService(userRepo, log)
	if err != nil {
		log.Fatalf("failed to initialize inboxuser service: %s", err)
	}

	timeBetweenOpenRequests := time.Minute * time.Duration(cfg.MinutesBetweenOpenRequests)
	timeBetweenClosedRequests := time.Minute * time.Duration(cfg.MinutesBetweenClosedRequests)
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
	metadataService := inbox.NewMetadataService(metadataRepo, providerService, credentialService, log, time.Duration(cfg.InboxMetadataTTLMinutes)*time.Minute)

	var classificationRepo classification.ClassificationRepository
	if cfg.ClassificationStore == "filesystem" {
		log.Println("Using filesystem classification repo")
		classificationRepo = classification.NewFileClassificationRepository(cfg.ClassificationFilesystemDirectory, cfg.ClassificationPageSize)
	} else {
		classificationRepo = classification.NewS3ClassificationRepository(s3, cfg.S3Bucket, cfg.ClassificationPageSize)
	}
	classificationService := classification.NewService(
		credentialService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		metadataService,
		userService,
		log,
		classifyQueuePublisher,
		cfg.ClassificationSubjectPreviews,
	)

	loginRepo := login.NewDynamoLinkRepository(db, cfg.DynamoDBTable)

	sessionRepo := session.NewDynamoSessionRepository(db, cfg.DynamoDBTable)
	if cfg.SessionDurationMinutes == 0 {
		log.Fatal("session duration is required")
	}
	sessionDuration := time.Minute * time.Duration(cfg.SessionDurationMinutes)
	sessionService := session.NewService(sessionRepo, cfg.SessionKeyLength, sessionDuration)

	var loginSender login.Sender
	var emailSender email.Sender
	if cfg.Env == "dev" {
		loginSender = login.NewLogSender(log, cfg.LoginBaseURL)
	} else if cfg.Env == "e2e" {
		if cfg.TestLoginServerSenderPort == 0 {
			log.Fatal("server started with env=e2e but no TestLoginServerSenderPort configured")
		}
		var testLoginServerSenderUsers []string
		if cfg.TestLoginServerSenderUsers == "" {
			log.Fatal("server started with env=e2e but no TestLoginServerSenderUser configured")
		}
		testLoginServerSenderUsers = strings.Split(cfg.TestLoginServerSenderUsers, ",")
		loginSender = login.NewTestServerLoginSender(cfg.TestLoginServerSenderPort, testLoginServerSenderUsers, cfg.LoginBaseURL, cfg.Env)
		log.Printf("using temporary login link server for E2E test environment and user %s", cfg.TestLoginServerSenderUsers)
	} else {
		sesClient, err := mailboxsweep.ConnectAWSService(mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.SESHost,
			ServicePort:   cfg.SESPort,
			ServiceRegion: cfg.SESRegion,
			Environment:   cfg.Env,
		}, sesv2.NewFromConfig)
		if err != nil {
			log.Fatalf("failed to initialize SES client: %s", err)
		}
		emailSender = email.NewSESSender(sesClient, cfg.EmailReplyToAddress, cfg.EmailSenderName, cfg.SESConfigSetName)
		loginSender = login.NewEmailSender(emailSender, cfg.SESLoginEmailTemplateName, cfg.LoginBaseURL)
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

	var userWhiteList []string
	if cfg.UserWhitelistConcat != nil {
		userWhiteList = strings.Split(*cfg.UserWhitelistConcat, ",")
	}
	loginService := login.NewService(loginLinkService, gmailLoginService, userService, sessionService, userWhiteList, log)

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
		commitQueuePublisher,
		log,
	)

	controller := httpserver.NewController(
		userService,
		credentialService,
		classifyRequestService,
		classificationService,
		metadataService,
		providerService,
		gmailLoginService,
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
	if cfg.CORSAllowedOriginsConcat == "" {
		log.Fatal("CORS_ALLOWED_ORIGIN is not configured")
	}
	corsOrigins := strings.Split(cfg.CORSAllowedOriginsConcat, ",")
	cors := httpserver.NewCORSHandler(corsOrigins)

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
			middleware.RealIP,
			middleware.RequestID,
			auth.Middleware,
			cors.Middleware,
			loggingMiddleware,
			// middleware.Heartbeat("/healthcheck"), // disabled for now because I don't want to make too many changes at once
			middleware.Recoverer,
			middleware.Timeout(time.Second * 10),
			middleware.AllowContentType("application/json"),
		},
	})

	if (cfg.Env == "prod" || cfg.Env == "staging") && cfg.XRayEnabled {
		srv = xray.Handler(xray.NewDynamicSegmentNamer("MailboxSweep", "api.*.mailboxsweep.com"), srv)
	}

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)

	log.Infof("Starting Mailboxsweep HTTP server at %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
