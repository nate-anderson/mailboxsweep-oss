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
	"github.com/nate-anderson/fig/v2"
)

type migrateConfig struct {
	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION"`
	DynamoDBTable  string  `fig:"DYNAMODB_TABLE"`

	S3Host   *string `fig:"S3_HOST"`
	S3Port   int     `fig:"S3_PORT"`
	S3Region string  `fig:"S3_REGION"`
	S3Bucket string  `fig:"S3_BUCKET"`

	SQSHost          *string `fig:"SQS_HOST"`
	SQSPort          int     `fig:"SQS_PORT"`
	SQSRegion        string  `fig:"SQS_REGION"`
	SQSClassifyQueue string  `fig:"SQS_CLASSIFY_QUEUE"`
	SQSClassifyDLQ   string  `fig:"SQS_CLASSIFY_DLQ"`
	SQSCommitQueue   string  `fig:"SQS_COMMIT_QUEUE"`
	SQSCommitDLQ     string  `fig:"SQS_COMMIT_DLQ"`

	Env string `fig:"ENV"`
}

func main() {
	cmd := os.Args[1]

	envDriver, err := fig.NewOptionalFileEnvironmentDriver(".env")
	if err != nil {
		log.Fatal(err)
	}
	conf := fig.New(envDriver)

	cfg := migrateConfig{}
	if err := conf.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	switch strings.ToLower(cmd) {
	case "dynamo":
		awsCfg := mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.DynamoDBHost,
			ServicePort:   cfg.DynamoDBPort,
			ServiceRegion: cfg.DynamoDBRegion,
			Environment:   cfg.Env,
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
		if cfg.Env != "dev" && cfg.Env != "e2e" {
			log.Fatal("cannot create bucket this way unless ENV in (dev|e2e)")
		}
		awsCfg := mailboxsweep.AWSServiceConfig{
			ServiceHost:   cfg.S3Host,
			ServicePort:   cfg.S3Port,
			ServiceRegion: cfg.S3Region,
			Environment:   cfg.Env,
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
				Environment:   cfg.Env,
			},
			sqs.NewFromConfig,
		)
		if err != nil {
			log.Fatal(err)
		}

		// create main classify FIFO queue
		res, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: &cfg.SQSClassifyQueue,
			Attributes: map[string]string{
				"FifoQueue": "true",
			},
		})
		if err != nil {
			log.Fatalf("failed to create main queue: %s\n", err)
		}

		log.Printf("Created main classify queue at %s (region %s)", *res.QueueUrl, cfg.S3Region)

		// classify DLQ
		res, err = sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: &cfg.SQSClassifyDLQ,
			Attributes: map[string]string{
				"FifoQueue": "true",
			},
		})
		if err != nil {
			log.Fatalf("failed to create DLQ: %s", err)
		}

		log.Printf("created classify DLQ at %s (region %s)\n", *res.QueueUrl, cfg.SQSRegion)

		// create main commit FIFO queue
		res, err = sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: &cfg.SQSCommitQueue,
			Attributes: map[string]string{
				"FifoQueue": "true",
			},
		})
		if err != nil {
			log.Fatalf("failed to create main queue: %s\n", err)
		}

		log.Printf("Created main commit queue at %s (region %s)", *res.QueueUrl, cfg.S3Region)

		// commit DLQ
		res, err = sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: &cfg.SQSCommitDLQ,
			Attributes: map[string]string{
				"FifoQueue": "true",
			},
		})
		if err != nil {
			log.Fatalf("failed to create DLQ: %s", err)
		}

		log.Printf("created commit DLQ at %s (region %s)\n", *res.QueueUrl, cfg.SQSRegion)

		// create FIFO DLQ
	default:
		log.Println("Unknown command " + cmd)
	}
}
