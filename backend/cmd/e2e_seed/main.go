package main

import (
	"context"
	"flag"
	"mailboxsweep"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/inboxuser"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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

	ctx := context.Background()
	if err := seedClassifiedTestUser(ctx, cfg, db, log); err != nil {
		panic(err)
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
