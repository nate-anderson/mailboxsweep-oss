package login

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GmailLoginService struct {
	oauthConfig *oauth2.Config
}

// OAuth2 scopes
var apiScopes = []string{
	"https://mail.google.com/",
}

func NewGmailLoginService(clientID, clientSecret, redirectURL string) *GmailLoginService {
	conf := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       apiScopes,
	}
	return &GmailLoginService{
		oauthConfig: &conf,
	}
}

func (s GmailLoginService) makeStateToken() (string, error) {
	buf := make([]byte, 24)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

func (s GmailLoginService) GetAuthURL() (string, error) {
	// @TODO: we need to check the state token when the user is redirected back here

	stateToken, err := s.makeStateToken()
	if err != nil {
		return "", err
	}

	return s.oauthConfig.AuthCodeURL(stateToken, oauth2.AccessTypeOffline), nil
}
