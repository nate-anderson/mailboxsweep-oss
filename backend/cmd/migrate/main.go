package main

import (
	"context"
	"fmt"
	"log"
	"mailboxsweep"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/viper"
)

type migrateConfig struct {
	DynamoDBHost   *string `mapstructure:"DYNAMODB_HOST"`
	DynamoDBPort   int     `mapstructure:"DYNAMODB_PORT"`
	DynamoDBRegion string  `mapstructure:"DYNAMODB_REGION"`
	DynamoDBTable  string  `mapstructure:"DYNAMODB_TABLE"`

	S3Host   *string `mapstructure:"S3_HOST"`
	S3Port   int     `mapstructure:"S3_PORT"`
	S3Region string  `mapstructure:"S3_REGION"`
	S3Bucket string  `mapstructure:"S3_BUCKET"`

	SQSHost   *string `mapstructure:"SQS_HOST"`
	SQSPort   int     `mapstructure:"SQS_PORT"`
	SQSRegion string  `mapstructure:"SQS_REGION"`
	SQSQueue  string  `mapstructure:"SQS_QUEUE"`
	SQSDLQ    string  `mapstructure:"SQS_DLQ"`

	Env string `mapstructure:"ENV"`
}

func main() {
	cmd := os.Args[1]

	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	cfg := migrateConfig{}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	switch strings.ToLower(cmd) {
	case "dynamo":
		awsCfg := mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.DynamoDBHost,
			ServicePort:   cfg.DynamoDBPort,
			ServiceRegion: cfg.DynamoDBRegion,
		}
		tableDef := mailboxsweep.CreateDynamoDBTableDefinition(cfg.DynamoDBTable)
		cl, err := mailboxsweep.ConnectAWSService(
			awsCfg,
			dynamodb.NewFromConfig,
		)
		if err != nil {
			log.Fatal(err)
		}

		_, err = cl.CreateTable(ctx, tableDef)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Created table %s!\n", cfg.DynamoDBTable)
	case "s3":
		if cfg.Env != "dev" {
			log.Fatal("cannot create bucket this way unless ENV == dev")
		}
		awsCfg := mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.S3Host,
			ServicePort:   cfg.S3Port,
			ServiceRegion: cfg.S3Region,
		}
		fmt.Printf("Initializing S3 with addr %s:%d (bucket %s)\n", *cfg.S3Host, cfg.S3Port, cfg.S3Bucket)

		cl, err := mailboxsweep.ConnectAWSService(
			awsCfg,
			s3.NewFromConfig,
			func(o *s3.Options) {
				o.UsePathStyle = true
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		_, err = cl.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: &cfg.S3Bucket,
			CreateBucketConfiguration: &types.CreateBucketConfiguration{
				LocationConstraint: "us-west-2",
			},
		})
		if err != nil {
			panic(err)
		}
		log.Printf("Created bucket %s!\n", cfg.S3Bucket)

	case "sqs":
		sqsClient, err := mailboxsweep.ConnectAWSService(
			mailboxsweep.AWSServiceConfig{
				ServiceHost:   cfg.SQSHost,
				ServicePort:   cfg.SQSPort,
				ServiceRegion: cfg.SQSRegion,
			},
			sqs.NewFromConfig,
		)
		if err != nil {
			log.Fatal(err)
		}

		// create main FIFO queue
		res, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: &cfg.SQSQueue,
			Attributes: map[string]string{
				"FifoQueue": "true",
			},
		})
		if err != nil {
			log.Fatalf("failed to create main queue: %s\n", err)
		}

		log.Printf("Created main queue at %s (region %s)", *res.QueueUrl, cfg.S3Region)

		res, err = sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: &cfg.SQSDLQ,
			Attributes: map[string]string{
				"FifoQueue": "true",
			},
		})
		if err != nil {
			log.Fatalf("failed to create DLQ: %s", err)
		}

		log.Printf("created DLQ at %s (region %s)\n", *res.QueueUrl, cfg.SQSRegion)

		// create FIFO DLQ
	default:
		log.Println("Unknown command " + cmd)
	}
}
