package login

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mailboxsweep"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/pkg/slice"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleoauth "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type GmailLoginService struct {
	oauthConfig       *oauth2.Config
	tokenRepo         GmailTokenRepository
	credentialService inboxcredential.Service
	userService       inboxuser.Service
}

func NewGmailLoginService(tokenRepo GmailTokenRepository, credentialService inboxcredential.Service, userService inboxuser.Service, clientID, clientSecret, redirectURL string) *GmailLoginService {
	conf := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       mailboxsweep.GoogleAPIScopes,
	}
	return &GmailLoginService{
		oauthConfig:       &conf,
		tokenRepo:         tokenRepo,
		credentialService: credentialService,
		userService:       userService,
	}
}

const tokenLen = 40

func (s GmailLoginService) makeToken() (string, error) {
	buf := make([]byte, tokenLen)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	return strings.ToLower(base64.URLEncoding.EncodeToString(buf)), nil
}

func (s GmailLoginService) GetAuthURL(ctx context.Context) (redirect string, cookie string, err error) {
	stateToken, err := s.makeToken()
	if err != nil {
		return "", "", err
	}

	cookie, err = s.makeToken()
	if err != nil {
		return "", "", err
	}

	token := GmailLoginToken{
		Token:  stateToken,
		Cookie: cookie,
	}

	err = s.tokenRepo.Store(ctx, token)
	if err != nil {
		return "", "", err
	}

	return s.oauthConfig.AuthCodeURL(stateToken, oauth2.AccessTypeOffline), cookie, nil
}

type GmailLoginRedirectData struct {
	// The state token we gave to Google when the user initiated login
	StateToken string
	// The cookie we set on the user's browser when they initiated login
	ExternalAuthCookie string
	// The temporary access code Google gives us in the user's redirect URL query
	Code string
}

// CompleteLogin takes the data from the redirect provided by Google, validates the user's external auth cookie
// and the token Google gives back, and returns the user's address who initiated the login
// If the user has not had their profile data pulled, we pull it and store it
func (s GmailLoginService) CompleteLogin(ctx context.Context, data GmailLoginRedirectData) (string, error) {
	known, err := s.tokenRepo.Get(ctx, data.StateToken)
	if err != nil {
		return "", err
	}

	authErr := errors.New("redirect data mismatch")

	if known.Cookie != strings.TrimSpace(data.ExternalAuthCookie) {
		return "", mailboxsweep.UnauthenticatedError(authErr)
	}

	if known.Token != strings.TrimSpace(data.StateToken) {
		return "", mailboxsweep.UnauthenticatedError(authErr)
	}

	// exchange code for real credentials
	token, err := s.oauthConfig.Exchange(ctx, data.Code)
	if err != nil {
		err := fmt.Errorf("failed to exchange with Google for login: %w", err)
		return "", mailboxsweep.BadGatewayError(err)
	}

	client := s.oauthConfig.Client(ctx, token)
	oauthService, err := googleoauth.NewService(ctx, option.WithScopes(mailboxsweep.GoogleAPIScopes...), option.WithHTTPClient(client))
	if err != nil {
		return "", err
	}

	info, err := oauthService.Tokeninfo().Do()
	if err != nil {
		return "", err
	}
	scopes := strings.Fields(info.Scope)
	for _, scope := range mailboxsweep.GoogleAPIScopes {
		if !slice.Contains(scopes, scope) {
			return "", mailboxsweep.InsufficientPermissionsError(fmt.Errorf("scope %s was not granted", scope))
		}
	}

	profile, err := s.fetchUserProfile(ctx, token)
	if err != nil {
		return "", err
	}

	// store the user
	user, err := s.userService.Store(ctx, profile.address)
	if err != nil {
		return "", err
	}

	// the user is authenticated, authorize their context
	authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, user.Address)

	err = s.userService.SetName(authedCtx, profile.nameFirst, profile.nameLast)
	if err != nil {
		return "", err
	}

	// store the Gmail credentials
	gmailCred := inboxcredential.NewGmailInboxCredential(profile.address, token.AccessToken, token.RefreshToken, token.Expiry)
	err = s.credentialService.StoreCredential(authedCtx, gmailCred)
	if err != nil {
		return "", fmt.Errorf("failed to store Gmail token: %w", err)
	}

	return profile.address, nil
}

type googleProfile struct {
	address   string
	nameFirst string
	nameLast  string
}

func (s GmailLoginService) fetchUserProfile(ctx context.Context, token *oauth2.Token) (*googleProfile, error) {
	client := s.oauthConfig.Client(ctx, token)

	oauthService, err := googleoauth.NewService(ctx, option.WithScopes(mailboxsweep.GoogleAPIScopes...), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res, err := oauthService.Userinfo.V2.Me.Get().Do()
	if err != nil {
		return nil, err
	}

	return &googleProfile{
		address:   res.Email,
		nameFirst: res.GivenName,
		nameLast:  res.FamilyName,
	}, err
}
