package email

import (
	"context"
	"encoding/json"
	"net/mail"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESSender struct {
	client         *sesv2.Client
	replyToAddress string
	senderName     string
}

func NewSESSender(sesClient *sesv2.Client, replyToAddress, senderName string) *SESSender {
	return &SESSender{
		client:         sesClient,
		replyToAddress: replyToAddress,
		senderName:     senderName,
	}
}

func (s SESSender) Send(ctx context.Context, address, template string, data map[string]string) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	dataStr := string(dataJSON)

	addr := mail.Address{
		Name:    s.senderName,
		Address: s.replyToAddress,
	}
	formattedAddress := addr.String()

	_, err = s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{address},
		},
		ReplyToAddresses: []string{formattedAddress},
		FromEmailAddress: &formattedAddress,
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: &template,
				TemplateData: &dataStr,
			},
		},
	})

	return err
}
