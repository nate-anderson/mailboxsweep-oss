package inboxuser

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

/*
Inbox user defines a single inbox "user" of the system - mailboxsweep
treats each inbox as a user and does not identify users across multiple
inboxes, if they use the tool on multiple accoutns
*/

type InboxUser struct {
	UUID       string                  `dynamodbav:"uuid"`
	Address    string                  `dynamodbav:"address"`
	NameFirst  string                  `dynamodbav:"name_first"`
	NameLast   string                  `dynamodbav:"name_last"`
	Payments   []Payment               `dynamodbav:"payments"`
	Created    time.Time               `dynamodbav:"created"`
	LastSeen   time.Time               `dynamodbav:"last_seen"`
	Agreements map[Agreement]time.Time `dynamodbav:"agreements"`
}

func NewInboxUser(address string) (*InboxUser, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	return &InboxUser{
		UUID:       id.String(),
		Address:    address,
		Created:    time.Now(),
		LastSeen:   time.Now(),
		Agreements: make(map[Agreement]time.Time),
		Payments:   []Payment{},
	}, nil
}

type Plan string

var (
	Basic  Plan = "BASIC"
	Deluxe Plan = "DELUXE"
)

func ParsePlan(str string) (Plan, error) {
	clean := strings.ToUpper(
		strings.TrimSpace(str),
	)

	switch Plan(clean) {
	case Basic:
		return Basic, nil
	case Deluxe:
		return Deluxe, nil
	default:
		return "", fmt.Errorf("unknown plan code '%s'", str)
	}
}

type PaymentMethod string

var (
	Stripe PaymentMethod = "STRIPE"
)

type Payment struct {
	Plan          Plan          `dynamodbav:"plan"`
	AmountInCents int           `dynamodbav:"amount_in_cents"`
	Method        PaymentMethod `dynamodbav:"payment_method"`
	ExternalID    string        `dynamodbav:"external_id"`
	Timestamp     time.Time     `dynamodbav:"timestamp"`
}

type Agreement string

var (
	TermsAndConditions Agreement = "TERMS_AND_CONDITIONS"
	PrivacyPolicy      Agreement = "PRIVACY_POLICY"
)

var requiredAgreements = []Agreement{
	TermsAndConditions,
}

var allAgreements = []Agreement{
	TermsAndConditions,
}

func ParseAgreement(str string) (Agreement, error) {
	switch Agreement(str) {
	case TermsAndConditions:
		return TermsAndConditions, nil
	default:
		return "", fmt.Errorf("unknown agreement string '%s'", str)
	}
}

func (u InboxUser) ProfileComplete() bool {
	return u.NameFirst != "" && u.NameLast != ""
}
