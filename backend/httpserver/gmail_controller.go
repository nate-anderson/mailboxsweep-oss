package httpserver

import (
	"mailboxsweep/login"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type GmailReauthController struct {
	baseController
	gmailLoginService *login.GmailLoginService
	cookieDomain      string
	cookiePath        string
}

func NewGmailReauthController(gmailLoginService *login.GmailLoginService, cookieDomain string, cookiePath string, log *logrus.Logger) *GmailReauthController {
	return &GmailReauthController{
		baseController:    baseController{log},
		cookieDomain:      cookieDomain,
		cookiePath:        cookieDomain,
		gmailLoginService: gmailLoginService,
	}
}

func (c GmailReauthController) ReauthenticateGmail(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	redirect, cookieVal, err := c.gmailLoginService.GetAuthURL(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	res := StartLoginResponse{
		Ok:          true,
		Redirect:    true,
		RedirectUrl: &redirect,
	}

	cookie := &http.Cookie{
		Name:     externalAuthCookieName,
		Value:    cookieVal,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(externalAuthCookieDuration),
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, cookie)
	c.respond(w, http.StatusOK, res)
}
