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
	"github.com/nate-anderson/fig/v2"
	"github.com/sirupsen/logrus"
)

type redriveConfig struct {
	SQSHost   *string `fig:"SQS_HOST"`
	SQSPort   int     `fig:"SQS_PORT"`
	SQSRegion string  `fig:"SQS_REGION"`
	QueueURL  string  `fig:"SQS_QUEUE_URL"`
	DLQURL    string  `fig:"SQS_DLQ_URL"`
}

func main() {
	envDriver, err := fig.NewEnvironmentDriver(".env")
	if err != nil {
		log.Fatal(err)
	}
	conf := fig.New(envDriver)

	cfg := redriveConfig{}
	if err := conf.Unmarshal(&cfg); err != nil {
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
				log.Fatalf("failed to remove message from queue after success: %v (error %s)", m, err)
			}
		}
	}
}
