package inbox

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/pkg/slice"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	googleoauth "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type GmailProvider struct {
	oauthConfig *oauth2.Config
	limits      *GmailAPIRateLimits
	logger      *logrus.Logger
}

const redirectURL = "mailboxsweep.com"

func NewGmailProvider(
	ctx context.Context,
	clientID string,
	clientSecret string,
	limits GmailAPIRateLimits,
	logger *logrus.Logger,
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
		limits:      &limits,
		logger:      logger,
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

	p.logger.Infof("credential refresh token empty? %v (length = %d, encrypted = %v)", gmailCred.RefreshToken == "", len(gmailCred.RefreshToken), gmailCred.Encrypted)
	oauthToken, err := gmailCred.OAuth2Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail OAuth2 token: %w", err)
	}

	// httpClient := p.oauthConfig.Client(ctx, oauthToken)

	tokenSource := oauth2.ReuseTokenSource(
		oauthToken,
		p.oauthConfig.TokenSource(ctx, oauthToken),
	)
	client, err := gmail.NewService(
		context.Background(),
		// option.WithHTTPClient(httpClient),
		option.WithTokenSource(tokenSource),
	)
	if err != nil {
		return nil, err
	}

	if err := p.verifyPermissions(ctx, tokenSource); err != nil {
		return nil, err
	}

	return &GmailInbox{
		address: username,
		client:  client,
		limits:  p.limits,
		logger:  p.logger,
	}, nil
}

func (p *GmailProvider) verifyPermissions(ctx context.Context, tokenSource oauth2.TokenSource) error {
	oauthService, err := googleoauth.NewService(
		ctx,
		option.WithScopes(mailboxsweep.FullGoogleAPIScopes...),
		option.WithTokenSource(tokenSource),
	)
	if err != nil {
		return fmt.Errorf("failed to initiate Oauth service for Google permission check: %w", err)
	}

	info, err := oauthService.Tokeninfo().Do()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") || strings.Contains(err.Error(), "token expired") {
			return mailboxsweep.GoogleReauthenticationRequiredError(err)
		}
		return fmt.Errorf("failed to verify Gmail credential permissions: %w", err)
	}

	scopes := strings.Fields(info.Scope)
	for _, scope := range mailboxsweep.FullGoogleAPIScopes {
		if !slice.Contains(scopes, scope) {
			return mailboxsweep.InsufficientPermissionsError(fmt.Errorf("required Gmail scope %s was not granted", scope))
		}
	}

	return nil
}

func (p *GmailProvider) WrapError(err error) error {
	return mailboxsweep.GmailError(err)
}
