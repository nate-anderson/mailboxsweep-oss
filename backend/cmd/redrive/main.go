package main

import (
	"context"
	"encoding/json"
	"log"
	"mailboxsweep"
	"mailboxsweep/classification"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type redriveConfig struct {
	SQSHost   *string `mapstructure:"SQS_HOST"`
	SQSPort   int     `mapstructure:"SQS_PORT"`
	SQSRegion string  `mapstructure:"SQS_REGION"`
	QueueURL  string  `mapstructure:"SQS_QUEUE_URL"`
	DLQURL    string  `mapstructure:"SQS_DLQ_URL"`
}

func main() {

	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	cfg := redriveConfig{}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

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

	log := logrus.New()

	pub := mailboxsweep.NewSQSEventPublisher[classification.ClassifyQueueItem](sqsClient, cfg.QueueURL)

	for {
		retrieveID, err := uuid.NewUUID()

		batch, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:                &cfg.DLQURL,
			ReceiveRequestAttemptId: aws.String(retrieveID.String()),
			WaitTimeSeconds:         30,
		})
		if err != nil {
			panic(err)
		}

		for _, m := range batch.Messages {
			log.Println("Got DLQ msg: ", *m.MessageId)
			queueItem := mailboxsweep.EventBody[classification.ClassifyQueueItem]{}
			err := json.Unmarshal([]byte(*m.Body), &queueItem)
			if err != nil {
				log.Errorf("failed to unmarshal queue item: %s", err)
				continue
			}

			err = pub.PublishEvent(ctx, queueItem)
			if err != nil {
				log.Errorf("failed to publish to main queue: %s", err)
				continue
			}

			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &cfg.DLQURL,
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				log.Fatal("failed to remove message from queue after success: %v (error %s)", m, err)
			}
		}
	}
}
