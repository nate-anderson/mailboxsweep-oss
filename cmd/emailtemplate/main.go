package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"mailboxsweep"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"gopkg.in/yaml.v2"
)

//go:embed templates/*
var templatesFS embed.FS

type Template struct {
	Name      string `yaml:"name"`
	Subject   string `yaml:"subject"`
	HTML      string `yaml:"html"`
	PlainText string `yaml:"plaintext"`
}

var dir = "templates"

func readTemplates() ([]Template, error) {
	dirEntries, err := templatesFS.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	templates := make([]Template, 0, len(dirEntries))
	for _, entry := range dirEntries {
		path := filepath.Join(dir, entry.Name())
		file, err := templatesFS.Open(path)
		if err != nil {
			return nil, err
		}

		tmpl := Template{}
		err = yaml.NewDecoder(file).Decode(&tmpl)
		if err != nil {
			return nil, err
		}

		templates = append(templates, tmpl)
	}

	return templates, nil
}

func syncTemplates(ctx context.Context, ses *sesv2.Client, templates []Template) error {
	for _, tmpl := range templates {
		// rate limit is once per second
		_, err := ses.GetEmailTemplate(ctx, &sesv2.GetEmailTemplateInput{
			TemplateName: &tmpl.Name,
		})
		if err != nil {
			dne := new(types.NotFoundException)
			if !errors.As(err, &dne) {
				panic(err)
			}

			// it doesn't exist, create it
			_, err := ses.CreateEmailTemplate(ctx, &sesv2.CreateEmailTemplateInput{
				TemplateName: &tmpl.Name,
				TemplateContent: &types.EmailTemplateContent{
					Subject: &tmpl.Subject,
					Html:    &tmpl.HTML,
					Text:    &tmpl.PlainText,
				},
			})
			if err != nil {
				log.Fatalf("failed to create new template %s: %s", tmpl.Name, err)
			}
			continue
		}

		time.Sleep(time.Second)

		_, err = ses.UpdateEmailTemplate(ctx, &sesv2.UpdateEmailTemplateInput{
			TemplateName: &tmpl.Name,
			TemplateContent: &types.EmailTemplateContent{
				Html:    &tmpl.HTML,
				Text:    &tmpl.PlainText,
				Subject: &tmpl.Subject,
			},
		})
		if err != nil {
			log.Fatalf("failed to update existing template %s: %s", tmpl.Name, err)
		}
	}
	return nil
}

func main() {
	ses, err := mailboxsweep.ConnectAWSService(mailboxsweep.AWSServiceConfig{}, sesv2.NewFromConfig)
	if err != nil {
		log.Fatalf("failed to initialize SES client: %s", err)
	}

	templates, err := readTemplates()
	if err != nil {
		log.Fatalf("failed to read templates from disk: %s", err)
	}

	ctx := context.Background()
	err = syncTemplates(ctx, ses, templates)
	if err != nil {
		log.Fatalf("sync failed: %s", err)
	}
	log.Println("Complete!")
}
