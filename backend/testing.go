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
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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
	token, err := randToken(5)
	if err != nil {
		log.Fatalf("failed to generate a test bucket name: %s", err)
	}
	bucket := strings.ToLower(t.Name() + "-" + token)

	client, err := ConnectAWSService(AWSServiceConfig{
		ServiceHost:   aws.String("localhost"),
		ServicePort:   4566,
		ServiceRegion: "us-west-2",
	},
		s3.NewFromConfig,
		func(ot *s3.Options) {
			ot.UsePathStyle = true
		},
	)

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

func NewTestSMTPSender(conf testConfig) *testSMTPSender {
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
	Address  string `mapstructure:"TEST_ADDRESS"`
	Password string `mapstructure:"TEST_PASSWORD"`
	IMAPHost string `mapstructure:"TEST_IMAP_HOST"`
	IMAPPort int    `mapstructure:"TEST_IMAP_PORT"`
	SMTPHost string `mapstructure:"TEST_SMTP_HOST"`
	SMTPPort int    `mapstructure:"TEST_SMTP_PORT"`
}

func GetTestConfig() (testConfig, error) {
	tc := testConfig{}
	viper.SetConfigType("env")
	viper.SetConfigFile("test.env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return tc, err
	}

	if err := viper.Unmarshal(&tc); err != nil {
		return tc, err
	}

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

	tableDef := CreateDynamoDBTableDefinition(table)

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
