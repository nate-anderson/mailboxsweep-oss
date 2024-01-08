package mailboxsweep

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nate-anderson/fig/v2"

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

func SetupTestBucket(ctx context.Context, t *testing.T) (*s3.Client, string, func()) {
	var host string
	// default to localhost if environment doesn't specify otherwise
	if host = os.Getenv("S3_HOST"); host == "" {
		host = "localhost"
	}

	token, err := randToken(5)
	if err != nil {
		log.Fatalf("failed to generate a test bucket name: %s", err)
	}
	bucket := strings.ToLower(t.Name() + "-" + token)

	client, err := ConnectAWSService(AWSServiceConfig{
		ServiceHost:   &host,
		ServicePort:   4566,
		ServiceRegion: "us-west-2",
	},
		s3.NewFromConfig,
		func(ot *s3.Options) {
			ot.UsePathStyle = true
		},
	)
	if err != nil {
		panic(err)
	}

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

	teardown := func() {
		objects, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: &bucket,
		})
		if err != nil {
			log.Fatalf("error listing test bucket '%s' contents to clear: %s", bucket, err)
		}
		for _, o := range objects.Contents {
			_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: &bucket,
				Key:    o.Key,
			})
			if err != nil {
				log.Fatalf("error deleting object %s from test bucket '%s': %s", *o.Key, bucket, err)
			}
		}
		_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: &bucket,
		})
		if err != nil {
			log.Printf("error tearing down test bucket '%s': %s", bucket, err)
		}
	}

	return client, bucket, teardown
}

type testSMTPSender struct {
	host   string
	addr   string
	auth   smtp.Auth
	client *smtp.Client
}

func NewTestSMTPSender(conf testConfig) (*testSMTPSender, error) {
	addr := net.JoinHostPort(conf.SMTPHost, strconv.Itoa(conf.SMTPPort))
	auth := smtp.PlainAuth("", conf.Address, conf.Password, conf.SMTPHost)
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		err := client.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
			ServerName:         conf.SMTPHost,
		})
		if err != nil {
			return nil, err
		}
	}

	if ok, _ := client.Extension("AUTH"); !ok {
		return nil, errors.New("smtp: server doesn't support AUTH")
	}
	if err = client.Auth(auth); err != nil {
		return nil, err
	}

	return &testSMTPSender{
		host:   conf.SMTPHost,
		addr:   addr,
		auth:   auth,
		client: client,
	}, nil
}

func (s testSMTPSender) formatMsg(to, from, subject, body string) string {
	header := map[string]string{
		"From":                      from,
		"To":                        to,
		"Subject":                   subject,
		"MIME-Version":              "1.0",
		"Content-Type":              "text/plain; charset=\"utf-8\"",
		"Content-Transfer-Encoding": "base64",
	}

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.RawStdEncoding.EncodeToString([]byte(body))
	return message
}

func (s testSMTPSender) Send(to, from, subj, body string) error {
	if err := s.client.Mail(from); err != nil {
		return err
	}
	if err := s.client.Rcpt(to); err != nil {
		return err
	}
	w, err := s.client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(s.formatMsg(to, from, subj, body))); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (s testSMTPSender) Close() error { return s.client.Close() }

type testConfig struct {
	Address  string `fig:"TEST_ADDRESS"`
	Password string `fig:"TEST_PASSWORD"`
	IMAPHost string `fig:"TEST_IMAP_HOST"`
	IMAPPort int    `fig:"TEST_IMAP_PORT"`
	SMTPHost string `fig:"TEST_SMTP_HOST"`
	SMTPPort int    `fig:"TEST_SMTP_PORT"`
}

func GetTestConfig() (testConfig, error) {
	tc := testConfig{}
	var conf fig.Config

	if _, err := os.Stat("test.env"); err == nil {
		driver, err := fig.NewEnvironmentDriver("test.env")
		if err != nil {
			panic(err)
		}
		conf = fig.New(driver)
	} else {
		driver, err := fig.NewEnvironmentDriver()
		if err != nil {
			panic(err)
		}
		conf = fig.New(driver)
	}

	if err := conf.Unmarshal(&tc); err != nil {
		return tc, err
	}

	return tc, nil
}

func SetupTestDynamoClient() *dynamodb.Client {
	var host string
	// default to localhost if environment doesn't specify otherwise
	if host = os.Getenv("DYNAMODB_HOST"); host == "" {
		host = "localhost"
	}

	port := 4566
	var err error
	if portStr := os.Getenv("DYNAMODB_PORT"); portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			log.Fatal(err)
		}
	}

	endpoint := fmt.Sprintf("http://%s:%d", host, port)
	region := "us-west-2"

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           endpoint,
			SigningRegion: region,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
	)

	if err != nil {
		log.Fatalf("Cannot load the AWS configs: %s", err)
	}

	return dynamodb.NewFromConfig(awsCfg)
}

func SetupDynamoTestTable(ctx context.Context, t *testing.T, client *dynamodb.Client) (table string, teardown func()) {
	t.Helper()

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

	return table, func() {
		_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: &table,
		})
		if err != nil {
			log.Printf("error tearing down test table %s: %s\n", table, err)
		}
	}
}
