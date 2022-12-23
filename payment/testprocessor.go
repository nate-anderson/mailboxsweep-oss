package payment

import (
	"mailboxsweep"

	"github.com/google/uuid"
)

type TestProcessor struct {
	sessions  map[string]CompletePayment
	sessionID string
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

var checkoutURL = "localhost:8080/checkout"

func (p *TestProcessor) InitiateCheckout(amount int64, email, name, description string) (checkoutURL string, sessionID string, err error) {
	id := uuid.NewString()
	if p.sessionID == "" {
		p.sessionID = id
	}
	p.sessions[p.sessionID] = CompletePayment{
		SessionID:    p.sessionID,
		TotalInCents: int(amount),
	}
	return checkoutURL, p.sessionID, nil
}
