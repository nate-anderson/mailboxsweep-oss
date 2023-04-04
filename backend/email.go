package mailboxsweep

import (
	"fmt"
	"net/mail"
	"strings"
)

// DomainFromEmail parses an email address and returns a lowercase domain
func DomainFromEmail(email string) (string, error) {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return "", fmt.Errorf("invalid email: %w", err)
	}

	tokens := strings.Split(email, "@")
	if len(tokens) < 2 {
		return "", fmt.Errorf("invalid email %s", email)
	}
	lastIdx := len(tokens) - 1
	domain := strings.ToLower(tokens[lastIdx])
	return domain, nil
}

const (
	DomainGmail = "gmail.com"
)
