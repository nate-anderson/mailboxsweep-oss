package inbox

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GmailProvider struct {
	client *gmail.Service
	conf   *oauth2.Config
}

const redirectURL = "mailboxsweep.com"

func NewGmailProvider(
	ctx context.Context,
	clientID string,
	clientSecret string,
) *GmailProvider {
	conf := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       mailboxsweep.GoogleAPIScopes,
	}

	return &GmailProvider{
		conf: &conf,
	}
}

func (p GmailProvider) Connect(_ context.Context) error {
	// no need
	return nil
}

func (p *GmailProvider) Login(ctx context.Context, username string, credential inboxcredential.Credential) (Inbox, error) {
	gmailCred, ok := credential.(*inboxcredential.GmailInboxCredential)
	if !ok {
		return nil, fmt.Errorf("expected Gmail credential: got %T", credential)
	}

	oauthToken, err := gmailCred.OAuth2Token()
	if err != nil {
		return nil, err
	}

	httpClient := p.conf.Client(ctx, oauthToken)

	client, err := gmail.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	p.client = client
	return &GmailInbox{
		address: username,
		client:  client,
	}, nil
}
