package login

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESSender struct {
	client       *sesv2.Client
	feedbackAddr string
	fromAddr     string
	templateName string
}

func NewSESSender(client *sesv2.Client, feedbackAddr string, fromAddr string, templateName string) *SESSender {
	return &SESSender{client, feedbackAddr, fromAddr, templateName}
}

func (s SESSender) Send(ctx context.Context, req SendRequest) error {
	tmplData := map[string]string{
		"signin_link": req.Token,
	}
	dataJSON, err := json.Marshal(tmplData)
	if err != nil {
		return err
	}

	_, err = s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FeedbackForwardingEmailAddress: &s.feedbackAddr,
		FromEmailAddress:               &s.fromAddr,
		Destination: &types.Destination{
			ToAddresses: []string{
				req.Address,
			},
		},
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: &s.templateName,
				TemplateData: aws.String(string(dataJSON)),
			},
		},
	})
	return err
}
