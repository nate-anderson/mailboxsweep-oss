package mailsweeper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/emersion/go-smtp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func randToken(l int) (string, error) {
	bytes := make([]byte, l)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func setupTestBucket(ctx context.Context, t *testing.T) (*s3.Client, string, func() error) {
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
		config.WithClientLogMode(aws.LogRequestWithBody|aws.LogResponseWithBody),
	)
	if err != nil {
		log.Fatalf("Cannot load the AWS configs: %s", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: "us-west-2",
		},
	})

	if err != nil {
		var errExist *types.BucketAlreadyOwnedByYou
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
}

func newTestSMTPSender(t *testing.T) *testSMTPSender {
	const (
		hostAddr = "localhost"
		hostPort = 3025
	)

	addr := net.JoinHostPort(hostAddr, strconv.Itoa(hostPort))

	return &testSMTPSender{
		host: hostAddr,
		addr: addr,
	}
}

func (s testSMTPSender) Send(to, from, subj, body string) error {
	return smtp.SendMail(s.addr, nil, from, []string{to}, strings.NewReader(body))
}
