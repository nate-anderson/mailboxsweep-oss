package email

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESSender struct {
	client         *sesv2.Client
	replyToAddress string
}

func NewSESSender(sesClient *sesv2.Client, replyToAddress string) *SESSender {
	return &SESSender{
		client:         sesClient,
		replyToAddress: replyToAddress,
	}
}

func (s SESSender) Send(ctx context.Context, address, template string, data map[string]string) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	dataStr := string(dataJSON)

	_, err = s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		ReplyToAddresses: []string{s.replyToAddress},
		FromEmailAddress: &s.replyToAddress,
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: &template,
				TemplateData: &dataStr,
			},
		},
	})

	return err
}
