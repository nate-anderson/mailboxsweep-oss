package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/inboxuser"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
)

type seedConfig struct {
	Env string `fig:"ENV" default:"prod"`

	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION" default:"us-west-2"`
	DynamoDBTable  string  `fig:"DYNAMODB_TABLE" required:"true"`

	ClassifyCompleteTestUser string `fig:"CLASSIFY_COMPLETE_TEST_USER" default:"test.classified@mailboxsweep.local"`

	ClassificationTestDataDir string  `fig:"CLASSIFICATION_TEST_DATA_DIR" default:".testdata/classifications"`
	S3Host                    *string `fig:"S3_HOST"`
	S3Port                    int     `fig:"S3_PORT"`
	S3Region                  string  `fig:"S3_REGION" default:"us-west-2"`
	ClassificationBucket      string  `fig:"CLASSIFICATION_BUCKET"`
	ClassificationPageSize    int     `fig:"CLASSIFICATION_PAGE_SIZE"`
}

func main() {
	log := logrus.New()

	// flags:
	// allow specifying configuration file
	var (
		envFile string
	)
	flag.StringVar(&envFile, "envFile", ".env", "specify .env configuration file to load")
	flag.Parse()

	envDriver, err := fig.NewOptionalFileEnvironmentDriver(envFile)
	if err != nil {
		log.Fatal(err)
	}

	// default to env then paramstore
	conf := fig.New(envDriver)

	cfg := seedConfig{}
	if err := conf.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := seedClassifiedTestUser(ctx, cfg, db, log); err != nil {
		log.Fatal(err)
	}

	if err := seedClassifications(ctx, cfg, s3, log); err != nil {
		log.Fatal(err)
	}
}

func seedClassifiedTestUser(ctx context.Context, cfg seedConfig, db *dynamodb.Client, log *logrus.Logger) error {
	userRepo := inboxuser.NewDynamoInboxUserRepository(db, cfg.DynamoDBTable)
	userService, err := inboxuser.NewService(userRepo, log)
	if err != nil {
		return err
	}

	user, err := userService.Store(ctx, cfg.ClassifyCompleteTestUser)
	if err != nil {
		return err
	}
	authedCtx := mailboxsweep.AuthorizeJobContext(ctx, cfg.ClassifyCompleteTestUser)

	log.Infof("Created user %s (%s)", user.Address, user.UUID)
	err = userService.SetName(authedCtx, "Test Classified", "User")
	if err != nil {
		return err
	}

	// consent to terms
	err = userService.Consent(authedCtx, []inboxuser.Agreement{inboxuser.TermsAndConditions, inboxuser.PrivacyPolicy})
	if err != nil {
		return err
	}

	// pay
	err = userService.StorePayment(authedCtx, inboxuser.Payment{
		Plan:          inboxuser.Basic,
		AmountInCents: 100,
		Method:        inboxuser.Stripe,
		ExternalID:    "aaa_test_123",
		Timestamp:     time.Now().Add(time.Hour * -1),
	})
	if err != nil {
		return err
	}

	// create classify request
	classifyRequestRepo := classifyrequest.NewDynamoRequestRepository(db, cfg.DynamoDBTable)
	classifyRequestService := classifyrequest.NewDefaultService(userService, classifyRequestRepo, time.Minute, time.Minute)

	if err = classifyRequestService.RecordRequest(authedCtx); err != nil {
		return err
	}

	// mark request completed
	if err = classifyRequestService.MarkCompletedSuccess(authedCtx); err != nil {
		return err
	}

	return nil
}

func seedClassifications(ctx context.Context, cfg seedConfig, s3 *s3.Client, log *logrus.Logger) error {
	files, err := os.ReadDir(cfg.ClassificationTestDataDir)
	if err != nil {
		return fmt.Errorf("failed to read test data dir %s: %w", cfg.ClassificationTestDataDir, err)
	}

	store := classification.NewS3ClassificationRepository(s3, cfg.ClassificationBucket, cfg.ClassificationPageSize)

	for _, fi := range files {
		fp := filepath.Join(cfg.ClassificationTestDataDir, fi.Name())
		file, err := os.Open(fp)
		if err != nil {
			return fmt.Errorf("failed to read test data file %s: %w", fp, err)
		}

		var c classification.ClassificationDocument
		err = json.NewDecoder(file).Decode(&c)
		if err != nil {
			return fmt.Errorf("failed to decode JSON from test data file %s: %w", fp, err)
		}

		err = store.Store(ctx, cfg.ClassifyCompleteTestUser, c.Classification, map[string]uint32{"INBOX": 0})
		if err != nil {
			return err
		}
	}

	return nil
}
