package mailboxsweep

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
)

type AWSServiceConfig struct {
	ServiceHost   *string
	ServicePort   int
	ServiceRegion string
	Environment   string
	XRayEnabled   bool
}

// Generic function for instantiating AWS clients
// the SDK is remarkably consistent
func ConnectAWSService[
	optType,
	clientType any,
	serviceConstructor func(cfg aws.Config, opts ...func(*optType)) *clientType,
](
	cfg AWSServiceConfig,
	constructor serviceConstructor,
	opts ...func(*optType),
) (*clientType, error) {
	if cfg.ServiceHost == nil {
		awsCfg, err := config.LoadDefaultConfig(
			context.Background(),
			config.WithRegion(cfg.ServiceRegion),
		)
		if (cfg.Environment == "prod" || cfg.Environment == "staging") && cfg.XRayEnabled {
			awsv2.AWSV2Instrumentor(&awsCfg.APIOptions)
		}
		if err != nil {
			return nil, err
		}

		return constructor(awsCfg), nil
	}

	endpoint := fmt.Sprintf("http://%s:%d", *cfg.ServiceHost, cfg.ServicePort)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           endpoint,
			SigningRegion: region,
		}, nil
	})

	// local environment
	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.ServiceRegion),
		config.WithEndpointResolverWithOptions(customResolver),
	)

	if err != nil {
		return nil, err
	}

	return constructor(awsCfg, opts...), nil
}
