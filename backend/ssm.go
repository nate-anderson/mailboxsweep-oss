package mailboxsweep

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/nate-anderson/fig/v2"
)

// SSMConfigDriver lets fig pull parameters from SSM
// This is very app specific. I
type SSMConfigDriver struct {
	ssmClient   *ssm.Client
	environment string
	prefix      string
}

func NewSSMConfigDriver(ssmClient *ssm.Client) *SSMConfigDriver {
	environment := os.Getenv("ENV")
	return &SSMConfigDriver{
		ssmClient:   ssmClient,
		environment: environment,
		prefix:      getSSMPrefix(environment),
	}
}

func getSSMPrefix(environment string) string {
	switch environment {
	case "prod":
		return "/mailboxsweep/PROD_"
	case "staging":
		return "/mailboxsweep/STAGING_"
	default:
		return ""
	}
}

func (s SSMConfigDriver) Get(key string) (string, error) {
	// use background context because config is only loaded at app startup,
	// there is not really a useful context to associate with this call
	result, err := s.ssmClient.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: ptr(true),
	})
	if err != nil {
		var notFound *types.ParameterNotFound
		if errors.As(err, &notFound) {
			return "", fig.ErrConfigNotFound
		}
		return "", err
	}

	return *result.Parameter.Value, nil
}

func (s SSMConfigDriver) Name() string {
	return "aws-paramstore-ssm"
}
