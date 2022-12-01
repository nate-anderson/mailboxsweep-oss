package mailboxsweep

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/joho/godotenv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func randToken(l int) (string, error) {
	bytes := make([]byte, l)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func SetupTestBucket(ctx context.Context, t *testing.T) (*s3.Client, string, func() error) {
	endpoint := "http://localhost:4566"
	region := "us-west-2"
	token, err := randToken(5)
	if err != nil {
		log.Fatalf("failed to generate a test bucket name: %s", err)
	}
	bucket := strings.ToLower(t.Name() + "-" + token)

	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           endpoint,
			SigningRegion: region,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithEndpointResolver(customResolver),
	)
	if err != nil {
		log.Fatalf("Cannot load the AWS configs: %s", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: "us-west-2",
		},
	})

	if err != nil {
		var errExist *s3types.BucketAlreadyOwnedByYou
		// if the bucket already exists, just use it
		// @TODO: revisit this later, test teardown should be cleaner
		if !errors.As(err, &errExist) {
			panic(err)
			// log.Fatalf("failed to create test bucket '%s': %s", bucket, err)
		}
	}

	teardown := func() error {
		_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: &bucket,
		})
		return err
	}

	return client, bucket, teardown
}

type testSMTPSender struct {
	host string
	addr string
	auth smtp.Auth
}

func newTestSMTPSender(conf testConfig) *testSMTPSender {
	addr := net.JoinHostPort(conf.SMTPHost, strconv.Itoa(conf.SMTPPort))
	auth := smtp.PlainAuth("", conf.Address, conf.Password, conf.SMTPHost)
	return &testSMTPSender{
		host: conf.SMTPHost,
		addr: addr,
		auth: auth,
	}
}

func (s testSMTPSender) Send(to, from, subj, body string) error {
	return smtp.SendMail(s.addr, s.auth, from, []string{to}, []byte(body))
}

type testConfig struct {
	Address  string
	Password string
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
}

func getTestConfig() (testConfig, error) {
	tc := testConfig{}
	if err := godotenv.Load("test.env"); err != nil {
		return tc, err
	}

	tc.Address = os.Getenv("TEST_ADDRESS")
	tc.IMAPHost = os.Getenv("TEST_IMAP_HOST")
	tc.SMTPHost = os.Getenv("TEST_SMTP_HOST")
	tc.Password = os.Getenv("TEST_PASSWORD")

	imapPort, err := strconv.Atoi(os.Getenv("TEST_IMAP_PORT"))
	if err != nil {
		return tc, err
	}
	tc.IMAPPort = imapPort

	smtpPort, err := strconv.Atoi(os.Getenv("TEST_SMTP_PORT"))
	if err != nil {
		return tc, err
	}
	tc.SMTPPort = smtpPort

	return tc, nil
}

func SetupTestDynamoClient() *dynamodb.Client {
	endpoint := "http://localhost:4566"
	region := "us-west-2"

	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           endpoint,
			SigningRegion: region,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithEndpointResolver(customResolver),
	)

	if err != nil {
		log.Fatalf("Cannot load the AWS configs: %s", err)
	}

	return dynamodb.NewFromConfig(awsCfg)
}

func SetupDynamoTestTable(ctx context.Context, t *testing.T, client *dynamodb.Client) (table string, teardown func() error) {
	tok, err := randToken(4)
	if err != nil {
		panic(fmt.Errorf("failed to generate table name: %w", err))
	}
	table = fmt.Sprintf("%s_%s", t.Name(), tok)

	tableDef := &dynamodb.CreateTableInput{
		TableName: &table,
		ProvisionedThroughput: &dynamoTypes.ProvisionedThroughput{
			ReadCapacityUnits:  ptr[int64](10),
			WriteCapacityUnits: ptr[int64](10),
		},
		AttributeDefinitions: []dynamoTypes.AttributeDefinition{
			{
				AttributeName: &DynamoPK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoSK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoGSI1PK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoGSI1SK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoLSI1SK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []dynamoTypes.KeySchemaElement{
			{
				AttributeName: &DynamoPK,
				KeyType:       dynamoTypes.KeyTypeHash,
			},
			{
				AttributeName: &DynamoSK,
				KeyType:       dynamoTypes.KeyTypeRange,
			},
		},
		GlobalSecondaryIndexes: []dynamoTypes.GlobalSecondaryIndex{
			{
				IndexName: &DynamoGSI1,
				KeySchema: []dynamoTypes.KeySchemaElement{
					{
						AttributeName: &DynamoGSI1PK,
						KeyType:       dynamoTypes.KeyTypeHash,
					},
					{
						AttributeName: &DynamoGSI1SK,
						KeyType:       dynamoTypes.KeyTypeRange,
					},
				},
				Projection: &dynamoTypes.Projection{
					ProjectionType: dynamoTypes.ProjectionTypeKeysOnly,
				},
				ProvisionedThroughput: &dynamoTypes.ProvisionedThroughput{
					ReadCapacityUnits:  ptr[int64](10),
					WriteCapacityUnits: ptr[int64](10),
				},
			},
		},
		LocalSecondaryIndexes: []dynamoTypes.LocalSecondaryIndex{
			{
				IndexName: &DynamoLSI1,
				KeySchema: []dynamoTypes.KeySchemaElement{
					{
						AttributeName: &DynamoPK,
						KeyType:       dynamoTypes.KeyTypeHash,
					},
					{
						AttributeName: &DynamoLSI1SK,
						KeyType:       dynamoTypes.KeyTypeRange,
					},
				},
				Projection: &dynamoTypes.Projection{
					ProjectionType: dynamoTypes.ProjectionTypeKeysOnly,
				},
			},
		},
	}

	_, err = client.CreateTable(ctx, tableDef)
	if err != nil {
		panic(err)
	}

	return table, func() error {
		_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: &table,
		})
		return err
	}
}
