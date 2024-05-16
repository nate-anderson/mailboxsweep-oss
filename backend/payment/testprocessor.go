package payment

import (
	"mailboxsweep"

	"github.com/google/uuid"
)

type TestProcessor struct {
	sessions  map[string]CompletePayment
	SessionID string
}

func NewTestProcessor() *TestProcessor {
	return &TestProcessor{
		sessions: make(map[string]CompletePayment),
	}
}

func (p TestProcessor) FindCheckoutSession(sessionID string) (*CompletePayment, error) {
	pmt, ok := p.sessions[sessionID]
	if !ok {
		return nil, mailboxsweep.ErrNotFound
	}
	return &pmt, nil
}

var TestCheckoutURL = "localhost:8080/checkout"

func (p *TestProcessor) InitiateCheckout(param CheckoutParams) (checkout *CheckoutData, err error) {
	id := uuid.NewString()
	if p.SessionID == "" {
		p.SessionID = id
	}
	total := int(param.RegularPriceInCents)
	if param.DiscountInCents != nil {
		total -= int(*param.DiscountInCents)
	}
	p.sessions[p.SessionID] = CompletePayment{
		SessionID:    p.SessionID,
		TotalInCents: total,
	}
	return &CheckoutData{
		SessionID:    p.SessionID,
		ClientSecret: "fake-client-secret",
		CheckoutURL:  TestCheckoutURL,
	}, nil
}
