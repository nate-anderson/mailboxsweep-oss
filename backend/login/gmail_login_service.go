package login

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"mailboxsweep"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GmailLoginService struct {
	oauthConfig *oauth2.Config
	tokenRepo   GmailTokenRepository
}

// OAuth2 scopes
var apiScopes = []string{
	"https://mail.google.com/",
}

func NewGmailLoginService(tokenRepo GmailTokenRepository, clientID, clientSecret, redirectURL string) *GmailLoginService {
	conf := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       apiScopes,
	}
	return &GmailLoginService{
		oauthConfig: &conf,
		tokenRepo:   tokenRepo,
	}
}

const tokenLen = 30

func (s GmailLoginService) makeToken() (string, error) {
	buf := make([]byte, tokenLen)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

func (s GmailLoginService) GetAuthURL(ctx context.Context, address string) (redirect string, cookie string, err error) {
	stateToken, err := s.makeToken()
	if err != nil {
		return "", "", err
	}

	cookie, err = s.makeToken()
	if err != nil {
		return "", "", err
	}

	token := GmailLoginToken{
		Address: address,
		Token:   stateToken,
		Cookie:  cookie,
	}

	err = s.tokenRepo.Store(ctx, token)
	if err != nil {
		return "", "", err
	}

	return s.oauthConfig.AuthCodeURL(stateToken, oauth2.AccessTypeOffline), cookie, nil
}

func (s GmailLoginService) Validate(ctx context.Context, token, cookie string) (string, error) {
	known, err := s.tokenRepo.Get(ctx, token)
	if err != nil {
		return "", err
	}

	authErr := errors.New("redirect data mismatch")

	if known.Cookie != strings.TrimSpace(cookie) {
		return "", mailboxsweep.UnauthenticatedError(authErr)
	}

	if known.Token != strings.TrimSpace(token) {
		return "", mailboxsweep.UnauthenticatedError(authErr)
	}

	return known.Address, nil
}
