package inboxuser

import (
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

type PaymentMethod string

var (
	Stripe PaymentMethod = "STRIPE"
)

type Payment struct {
	Plan          Plan          `dynamodbav:"plan"`
	AmountInCents int           `dynamodbav:"amount_in_cents"`
	Method        PaymentMethod `dynamodbav:"payment_method"`
	Timestamp     time.Time     `dynamodbav:"timestamp"`
}

type Agreement string

var (
	PrivacyPolicy        Agreement = "PRIVACY_POLICY"
	NonCorporateEmail    Agreement = "NON_CORPORATE_EMAIL"
	BackupResponsibility Agreement = "BACKUP_RESPONSIBILITY"
	NecessaryCookies     Agreement = "COOKIES_NECESSARY"
)

var requiredAgreements = []Agreement{
	PrivacyPolicy,
	NonCorporateEmail,
	BackupResponsibility,
	NecessaryCookies,
}
