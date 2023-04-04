package login

import (
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

const redirectURL = "mailboxsweep.com"

func NewGmailLoginService(clientID, clientSecret, redirectURL string) *GmailLoginService {
	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       apiScopes,
	}
	return &GmailLoginService{
		&config,
	}
}

func (s GmailLoginService) GetAuthURL() string {
	return s.oauthConfig.AuthCodeURL("@TODO", oauth2.AccessTypeOffline)
}
