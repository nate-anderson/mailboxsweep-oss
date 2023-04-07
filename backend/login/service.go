package login

import (
	"context"
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

func (s Service) InitiateLogin(ctx context.Context, address, initiatedByIP string) (redirectURL string, redirect bool, cookie *string, err error) {
	domain, err := mailboxsweep.DomainFromEmail(address)
	if err != nil {
		return "", false, nil, err
	}

	switch domain {
	case mailboxsweep.DomainGmail:
		authURL, cookie, err := s.gmailLoginService.GetAuthURL(ctx, address)
		if err != nil {
			return "", false, nil, err
		}
		return authURL, true, &cookie, nil

	default:
		_, err := s.linkService.CreateLoginLink(ctx, address, initiatedByIP)
		if err != nil {
			return "", false, nil, err
		}
		return "", false, nil, nil
	}

}

func (s Service) ValidateLoginLink(ctx context.Context, token, signature string) (*inboxuser.InboxUser, *session.Session, error) {
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

func (s Service) ValidateGmailStateToken(ctx context.Context, token string, cookie string) (*inboxuser.InboxUser, *session.Session, error) {
	addr, err := s.gmailLoginService.Validate(ctx, token, cookie)
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

	return user, sess, nil
}
