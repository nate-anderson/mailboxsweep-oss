package inbox

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/pkg/slice"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	googleoauth "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type GmailProvider struct {
	oauthConfig *oauth2.Config
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
		Scopes:       mailboxsweep.FullGoogleAPIScopes,
	}

	return &GmailProvider{
		oauthConfig: &conf,
	}
}

func (p GmailProvider) Connect(_ context.Context) error {
	// no need
	return nil
}

// Login returns an InsufficientPermissionsError if there are missing required scopes
func (p *GmailProvider) Login(ctx context.Context, username string, credential inboxcredential.Credential) (Inbox, error) {
	gmailCred, ok := credential.(*inboxcredential.GmailInboxCredential)
	if !ok {
		return nil, fmt.Errorf("expected Gmail credential: got %T", credential)
	}

	oauthToken, err := gmailCred.OAuth2Token()
	if err != nil {
		return nil, err
	}

	httpClient := p.oauthConfig.Client(ctx, oauthToken)

	client, err := gmail.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	if err := p.verifyPermissions(ctx, oauthToken, httpClient); err != nil {
		return nil, err
	}

	return &GmailInbox{
		address: username,
		client:  client,
	}, nil
}

func (p *GmailProvider) verifyPermissions(ctx context.Context, token *oauth2.Token, client *http.Client) error {
	oauthService, err := googleoauth.NewService(ctx, option.WithScopes(mailboxsweep.FullGoogleAPIScopes...), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to initiate Oauth service for Google permission check: %w", err)
	}

	info, err := oauthService.Tokeninfo().Do()
	if err != nil {
		if strings.Contains(err.Error(), `invalid_grant`) {
			return mailboxsweep.GoogleReauthenticationRequiredError(err)
		}
		return err
	}

	scopes := strings.Fields(info.Scope)
	for _, scope := range mailboxsweep.FullGoogleAPIScopes {
		if !slice.Contains(scopes, scope) {
			return mailboxsweep.InsufficientPermissionsError(fmt.Errorf("required Gmail scope %s was not granted", scope))
		}
	}

	return nil
}
