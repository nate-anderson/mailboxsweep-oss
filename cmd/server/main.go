package main

import (
	"encoding/base64"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/httpserver"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type serverConfig struct {
	HTTPPort int `mapstructure:"HTTP_PORT"`

	DynamoDBHost   *string `mapstructure:"DYNAMODB_HOST"`
	DynamoDBPort   int     `mapstructure:"DYNAMODB_PORT"`
	DynamoDBRegion string  `mapstructure:"DYNAMODB_REGION"`

	S3Host   *string `mapstructure:"S3_HOST"`
	S3Port   int     `mapstructure:"S3_PORT"`
	S3Region string  `mapstructure:"S3_REGION"`

	EventBridgeHost   *string `mapstructure:"EVENTBRIDGE_HOST"`
	EventBridgePort   int     `mapstructure:"EVENTBRIDGE_PORT"`
	EventBridgeRegion string  `mapstructure:"EVENTBRIDGE_REGION"`

	DynamoDBTable             string `mapstructure:"DYNAMODB_TABLE"`
	S3Bucket                  string `mapstructure:"S3_BUCKET"`
	EventBusName              string `mapstructure:"EVENT_BUS_NAME"`
	CredentialCipherKeyBase64 string `mapstructure:"CREDENTIAL_CIPHER_KEY_BASE64"`
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

	imapServerRepo := imapserver.NewConstantIMAPServerRepository()
	imapServerService := imapserver.NewService(imapServerRepo)
	providerService := inbox.NewProviderService(imapServerService)

	bridge, err := mailboxsweep.ConnectAWSService(
		mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.EventBridgeHost,
			ServicePort:   cfg.EventBridgePort,
			ServiceRegion: cfg.EventBridgeRegion,
		},
		eventbridge.NewFromConfig,
	)

	classificationRepo := classification.NewS3ClassificationRepository(s3, cfg.S3Bucket)
	classificationService := classification.NewService(
		credentialService,
		imapServerService,
		classificationRepo,
		providerService,
		bridge,
		cfg.EventBusName,
	)

	folderService := inbox.NewFolderService(providerService, credentialService)

	controller := httpserver.NewController(
		credentialService,
		classificationService,
		folderService,
		log,
	)
	root := chi.NewMux()
	srv := httpserver.HandlerFromMux(controller, root)

	addr := ":8080"

	log.Infof("Starting Mailboxsweep HTTP server at %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
