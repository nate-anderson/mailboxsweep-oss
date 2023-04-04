package login

import (
	"context"
	"mailboxsweep"
)

type Service struct {
	linkService       LinkService
	gmailLoginService *GmailLoginService
}

func (s Service) InitiateLogin(ctx context.Context, address, initiatedByIP string) (redirectURL string, redirect bool, err error) {
	domain, err := mailboxsweep.DomainFromEmail(address)
	if err != nil {
		return "", false, err
	}

	switch domain {
	case mailboxsweep.DomainGmail:
		authURL := s.gmailLoginService.GetAuthURL()
		return authURL, true, nil

	default:
		_, err := s.linkService.CreateLoginLink(ctx, address, initiatedByIP)
		if err != nil {
			return "", false, err
		}
		return "", false, nil
	}

}
