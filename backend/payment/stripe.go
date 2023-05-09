package payment

import (
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/client"
)

type Processor interface {
	InitiateCheckout(amount int64, email, name, description string) (checkoutURL string, sessionID string, err error)
	FindCheckoutSession(sessionID string) (*CompletePayment, error)
}

type StripeProcessor struct {
	client             *client.API
	successURL         string
	checkoutExpiration time.Duration
}

func NewStripeProcessor(client *client.API, successURL string, checkoutExpiration time.Duration) (*StripeProcessor, error) {
	if checkoutExpiration == 0 {
		return nil, errors.New("stripe processor checkout session duration must be > 0")
	}
	return &StripeProcessor{
		client:             client,
		successURL:         successURL,
		checkoutExpiration: checkoutExpiration,
	}, nil
}

func ptr[t any](v t) *t {
	return &v
}

func (p StripeProcessor) InitiateCheckout(amount int64, email, name, description string) (checkoutURL string, sessionID string, err error) {
	sess, err := p.client.CheckoutSessions.New(&stripe.CheckoutSessionParams{
		AllowPromotionCodes: ptr(false),
		SuccessURL:          &p.successURL,
		// NV does not charge sales tax for digital products
		// https://tax.nv.gov/FAQs/Sales_Tax_Information___FAQs/
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: ptr(false),
		},
		Currency:      ptr("usd"),
		CustomerEmail: &email,
		Mode:          ptr("payment"),
		CustomText: &stripe.CheckoutSessionCustomTextParams{
			Submit: &stripe.CheckoutSessionCustomTextSubmitParams{
				Message: ptr("MAILBOXSWEEP CUSTOM TEXT"),
			},
		},
		ExpiresAt: ptr(time.Now().Add(p.checkoutExpiration).Unix()),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: ptr("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        &name,
						Description: &description,
					},
					UnitAmount:  ptr(amount),
					TaxBehavior: ptr("inclusive"),
				},
				Quantity: ptr[int64](1),
			},
		},
	})

	if err != nil {
		return "", "", err
	}

	return sess.URL, sess.ID, nil
}

type CompletePayment struct {
	SessionID    string
	TotalInCents int
}

func (p StripeProcessor) FindCheckoutSession(sessionID string) (*CompletePayment, error) {
	sess, err := p.client.CheckoutSessions.Get(sessionID, &stripe.CheckoutSessionParams{})
	if err != nil {
		return nil, err
	}

	if sess.Status != "complete" {
		return nil, fmt.Errorf("session %s is in status %s", sessionID, sess.Status)
	}

	return &CompletePayment{
		SessionID:    sess.ID,
		TotalInCents: int(sess.AmountTotal),
	}, nil
}
