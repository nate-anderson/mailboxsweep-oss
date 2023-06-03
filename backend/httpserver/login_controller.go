package httpserver

import (
	"encoding/json"
	"mailboxsweep/login"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LoginController struct {
	log          *logrus.Logger
	service      *login.Service
	cookieDomain string
	cookiePath   string
	isDev        bool
	baseController
}

func NewLoginController(service *login.Service, cookieDomain, cookiePath string, isDev bool, log *logrus.Logger) *LoginController {
	return &LoginController{
		log,
		service,
		cookieDomain,
		cookiePath,
		isDev,
		baseController{log},
	}
}

const externalAuthCookieName = "X-Mailboxsweep-Ext-Auth"
const externalAuthCookieDuration = time.Duration(15) * time.Minute

func (c LoginController) InitiateLogin(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := LoginRequest{}
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	// slow down login spamming
	time.Sleep(time.Second)

	loginInit, err := c.service.InitiateLogin(ctx, reqBody.Address, requestIP(req))
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	if loginInit.ExternalAuthCookie != nil {
		cookie := http.Cookie{
			Name:     externalAuthCookieName,
			Value:    *loginInit.ExternalAuthCookie,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			Expires:  time.Now().Add(externalAuthCookieDuration),
		}
		http.SetCookie(w, &cookie)
	}

	resBody := StartLoginResponse{
		Ok:          true,
		Redirect:    loginInit.Redirect,
		RedirectUrl: &loginInit.RedirectURL,
	}
	c.respond(w, http.StatusOK, resBody)
}

func (c LoginController) SubmitLoginToken(w http.ResponseWriter, req *http.Request, token, signature string) {
	ctx := req.Context()
	token = strings.TrimSpace(
		strings.ToLower(
			token,
		),
	)
	signature = strings.TrimSpace(
		strings.ToLower(
			signature,
		),
	)

	// slow down login spamming
	time.Sleep(time.Second)

	user, session, err := c.service.CompleteLinkLogin(ctx, token, signature)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	resBody := LoginResponse{
		Address:         user.Address,
		ProfileComplete: user.ProfileComplete(),
	}

	sessionCookie := session.AuthCookie(c.isDev, c.cookieDomain)
	http.SetCookie(w, sessionCookie)

	isLoggedInCookie := session.IsLoggedInCookie(c.cookieDomain)
	http.SetCookie(w, isLoggedInCookie)

	c.respond(w, http.StatusOK, resBody)
}

func (c LoginController) SubmitGmailStateToken(w http.ResponseWriter, req *http.Request, params SubmitGmailStateTokenParams) {
	ctx := req.Context()
	token := strings.TrimSpace(
		strings.ToLower(
			params.StateToken,
		),
	)

	cookie, err := req.Cookie(externalAuthCookieName)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	redirectData := login.GmailLoginRedirectData{
		StateToken:         token,
		ExternalAuthCookie: cookie.Value,
		Code:               params.Code,
	}
	user, sess, err := c.service.CompleteGmailLogin(ctx, redirectData)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	resBody := LoginResponse{
		Address: user.Address,

		// login.Service.CompleteGmailLogin pulls the user's profile data
		ProfileComplete: true,
	}

	sessionCookie := sess.AuthCookie(c.isDev, c.cookieDomain)
	http.SetCookie(w, sessionCookie)

	isLoggedInCookie := sess.IsLoggedInCookie(c.cookieDomain)
	http.SetCookie(w, isLoggedInCookie)

	c.respond(w, http.StatusOK, resBody)
}

func (c LoginController) Logout(w http.ResponseWriter, req *http.Request) {
	clearSessionCookie, clearLoggedInCookie, err := c.service.GenerateEndSessionCookie(req.Context(), c.cookieDomain)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	http.SetCookie(w, clearSessionCookie)
	http.SetCookie(w, clearLoggedInCookie)
}
