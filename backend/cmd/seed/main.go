package main

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/imapserver"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
)

/*
These are production data seeds, things that need to be in the DB after deployment
NOT seeds used for testing
*/

type seedCmdConfig struct {
	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION"`
	DynamoDBTable  string  `fig:"DYNAMODB_TABLE"`
}

func main() {
	log := logrus.New()

	envDriver, err := fig.NewEnvironmentDriver(".env")
	if err != nil {
		log.Fatal(err)
	}

	conf := fig.New(envDriver)

	cfg := seedCmdConfig{}
	if err := conf.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	imapServerSeeds := []imapserver.Server{
		{
			Domain:   "gmail.com",
			Host:     "imap.gmail.com",
			IsGlobal: true,
			Port:     993,
		},
		{
			Domain:   "yahoo.com",
			Host:     "export.imap.mail.yahoo.com",
			IsGlobal: true,
			Port:     993,
		},
		{
			Domain:   "ymail.com",
			Host:     "export.imap.mail.yahoo.com",
			IsGlobal: true,
			Port:     993,
		},
		{
			Domain:   "aol.com",
			Host:     "export.imap.aol.com",
			IsGlobal: true,
			Port:     993,
		},
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

	imapServerRepo := imapserver.NewIMAPServerRepository(db, cfg.DynamoDBTable)

	ctx := context.Background()
	for _, seed := range imapServerSeeds {
		err = imapServerRepo.Store(ctx, seed)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Done!")
}
