package login

import (
	"context"
	"errors"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"mailboxsweep/pkg/slice"
	"mailboxsweep/session"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Service struct {
	linkService       LinkService
	gmailLoginService *GmailLoginService
	userService       inboxuser.Service
	sessionService    session.Service

	log *logrus.Logger
}

func NewService(linkService LinkService, gmailLoginService *GmailLoginService, userService inboxuser.Service, sessionService session.Service, log *logrus.Logger) *Service {
	return &Service{
		linkService:       linkService,
		gmailLoginService: gmailLoginService,
		userService:       userService,
		sessionService:    sessionService,
		log:               log,
	}
}

type LoginInitResult struct {
	RedirectURL        string
	Redirect           bool
	ExternalAuthCookie *string
}

func (s Service) InitiateLogin(ctx context.Context, address, initiatedByIP string) (LoginInitResult, error) {
	var res LoginInitResult

	// Temporary login block for Staging environment
	if os.Getenv("ENV") == "staging" {
		allowedAddressesStr := os.Getenv("STAGING_ALLOWED_USERS")
		allowdAddresses := strings.Split(allowedAddressesStr, ",")
		if !slice.Contains(allowdAddresses, address) {
			return res, mailboxsweep.UnauthenticatedError(errors.New("user is not in test user list"))
		}
	}

	log := s.log.
		WithField("address", address).
		WithField("initiated_by_ip", initiatedByIP)

	domain, err := mailboxsweep.DomainFromEmail(address)
	if err != nil {
		log.Error("failed to determine domain from email", err)
		return res, err
	}

	log = log.WithField("login_domain", domain)

	// check if we know this user
	user, err := s.userService.FindByAddress(ctx, address)
	if err != nil && !errors.Is(err, mailboxsweep.ErrNotFound) {
		log.Errorf("non-404 user lookup error: %s", err)
		return res, nil
	}

	switch domain {
	case mailboxsweep.DomainGmail:
		if user != nil {
			// returning logins use magic link
			err := s.linkService.CreateLoginLink(ctx, *user, initiatedByIP)
			return res, err
		}
		authURL, cookie, err := s.gmailLoginService.GetAuthURL(ctx)
		if err != nil {
			return res, err
		}
		res.Redirect = true
		res.RedirectURL = authURL
		res.ExternalAuthCookie = &cookie
		log.Info("Google OAuth login initiated")
		return res, nil

	default:
		if user == nil {
			user, err = s.userService.Store(ctx, address)
			if err != nil {
				return res, err
			}
		}
		err := s.linkService.CreateLoginLink(ctx, *user, initiatedByIP)
		if err != nil {
			return res, err
		}
		log.Info("Magic Link login initiated")
		return res, nil
	}

}

func (s Service) CompleteLinkLogin(ctx context.Context, token, signature string) (*inboxuser.InboxUser, *session.Session, error) {
	userUUID, err := s.linkService.ValidateLoginLink(ctx, token, signature)
	if err != nil {
		s.log.Error("magic link login failed: ", err)
		return nil, nil, err
	}

	log := s.log.WithField("user_uuid", userUUID)

	user, err := s.userService.FindByUUID(ctx, userUUID)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	log = log.WithField("address", user.Address)

	sess, err := s.sessionService.StartSession(ctx, user.Address)
	if err != nil {
		log.Error("failed to start session: ", err)
		return nil, nil, err
	}

	log = log.WithField("session_exp", time.Unix(sess.Expires, 0).UTC().String())
	log.Info("magic link login completed, session started")
	return user, sess, nil
}

func (s Service) CompleteGmailLogin(ctx context.Context, loginData GmailLoginRedirectData) (*inboxuser.InboxUser, *session.Session, error) {
	addr, err := s.gmailLoginService.CompleteLogin(ctx, loginData)
	if err != nil {
		s.log.Error("failed to complete Gmail login: ", err)
		return nil, nil, err
	}

	log := s.log.
		WithField("address", addr).
		WithField("login_domain", "gmail.com")

	user, err := s.userService.FindByAddress(ctx, addr)
	if err != nil {
		log.Error("user not found", err)
		return nil, nil, err
	}

	sess, err := s.sessionService.StartSession(ctx, user.Address)
	if err != nil {
		log.Error("failed to start session: ", err)
		return nil, nil, err
	}

	log = log.WithField("session_exp", time.Unix(sess.Expires, 0).UTC().String())
	log.Info("Gmail login completed, session started")
	return user, sess, nil
}
