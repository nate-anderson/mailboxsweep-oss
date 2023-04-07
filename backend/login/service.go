package login

import (
	"context"
	"errors"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"mailboxsweep/session"
)

type Service struct {
	linkService       LinkService
	gmailLoginService *GmailLoginService
	userService       inboxuser.Service
	sessionService    session.Service
}

func NewService(linkService LinkService, gmailLoginService *GmailLoginService, userService inboxuser.Service, sessionService session.Service) *Service {
	return &Service{
		linkService:       linkService,
		gmailLoginService: gmailLoginService,
		userService:       userService,
		sessionService:    sessionService,
	}
}

type LoginInitResult struct {
	RedirectURL        string
	Redirect           bool
	ExternalAuthCookie *string
}

func (s Service) InitiateLogin(ctx context.Context, address, initiatedByIP string) (LoginInitResult, error) {
	var res LoginInitResult

	// check if we know this user
	user, err := s.userService.FindByAddress(ctx, address)
	if err != nil {
		// if this is an error other than "not found", fail
		if !errors.Is(err, mailboxsweep.ErrNotFound) {
			return res, err
		}

		// store the new user
		user, err = s.userService.Store(ctx, address)
		if err != nil {
			return res, err
		}

		// otherwise, consider this first login
	}

	domain, err := mailboxsweep.DomainFromEmail(address)
	if err != nil {
		return res, err
	}

	switch domain {
	case mailboxsweep.DomainGmail:
		authURL, cookie, err := s.gmailLoginService.GetAuthURL(ctx, user.Address)
		if err != nil {
			return res, err
		}
		res.Redirect = true
		res.RedirectURL = authURL
		res.ExternalAuthCookie = &cookie
		return res, nil

	default:
		err := s.linkService.CreateLoginLink(ctx, *user, initiatedByIP)
		if err != nil {
			return res, err
		}
		return res, nil
	}

}

func (s Service) CompleteLinkLogin(ctx context.Context, token, signature string) (*inboxuser.InboxUser, *session.Session, error) {
	userUUID, err := s.linkService.ValidateLoginLink(ctx, token, signature)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userService.FindByUUID(ctx, userUUID)
	if err != nil {
		return nil, nil, err
	}

	sess, err := s.sessionService.StartSession(ctx, user.Address)
	if err != nil {
		return nil, nil, err
	}

	return user, sess, nil
}

func (s Service) CompleteGmailLogin(ctx context.Context, loginData GmailLoginRedirectData) (*inboxuser.InboxUser, *session.Session, error) {
	addr, err := s.gmailLoginService.CompleteLogin(ctx, loginData)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userService.FindByAddress(ctx, addr)
	if err != nil {
		return nil, nil, err
	}

	sess, err := s.sessionService.StartSession(ctx, user.Address)
	if err != nil {
		return nil, nil, err
	}

	// fetch user's profile data if it's incomplete
	if !user.ProfileComplete() {
		// the user is authenticated, authorize their context
		authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, user.Address)
		nameFirst, nameLast, err := s.gmailLoginService.FetchUserProfile(authedCtx)
		if err != nil {
			return nil, nil, err
		}

		if err = s.userService.SetName(authedCtx, nameFirst, nameLast); err != nil {
			return nil, nil, err
		}
	}

	return user, sess, nil
}
