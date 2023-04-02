package login

type Service struct {
	linkService LinkService
}

// func (s Service) InitiateLogin(ctx context.Context, address, initiatedByIP string) (redirectURL string, redirect bool, err error) {
// 	domain, err := mailboxsweep.DomainFromEmail(address)
// 	if err != nil {
// 		return "", false, err
// 	}

// 	switch domain {
// 	case "gmail.com":

// 	default:
// 		_, err := s.linkService.CreateLoginLink(ctx, address, initiatedByIP)
// 		if err != nil {
// 			return "", false, err
// 		}
// 		return "", false, nil
// 	}

// }
