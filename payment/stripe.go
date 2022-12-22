package payment

import (
	"time"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/client"
)

type Processor interface {
	InitiateCheckout(amount int64, email, name, description string) (string, error)
}

type StripeProcessor struct {
	client             *client.API
	checkoutExpiration time.Duration
}

func ptr[t any](v t) *t {
	return &v
}

func (p StripeProcessor) InitiateCheckout(amount int64, email, name, description string) (string, error) {
	sess, err := p.client.CheckoutSessions.New(&stripe.CheckoutSessionParams{
		AllowPromotionCodes: ptr(false),
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: ptr(true),
		},
		Currency:      ptr("usd"),
		CustomerEmail: &email,
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
					UnitAmount: ptr(amount),
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return sess.URL, nil
}
