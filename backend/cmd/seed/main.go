package main

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/imapserver"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type seedCmdConfig struct {
	DynamoDBHost   *string `mapstructure:"DYNAMODB_HOST"`
	DynamoDBPort   int     `mapstructure:"DYNAMODB_PORT"`
	DynamoDBRegion string  `mapstructure:"DYNAMODB_REGION"`
	DynamoDBTable  string  `mapstructure:"DYNAMODB_TABLE"`
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

	cfg := seedCmdConfig{}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	imapServerSeeds := []imapserver.Server{
		{
			Domain: "gmail.com",
			Host:   "imap.gmail.com",
			Port:   993,
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
