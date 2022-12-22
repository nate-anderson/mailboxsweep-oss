package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"time"
)

type PaymentInfo struct {
	Plan       inboxuser.Plan
	ValidUntil time.Time
}

type Service interface {
	// returns the active user's plan if they have paid, else nil and ErrNeedsPayment
	VerifyPayment(ctx context.Context) (*PaymentInfo, error)
	// returns the checkout redirect URL
	// returns ErrAlreadyPaid if a payment is current
	InitiatePayment(ctx context.Context, plan inboxuser.Plan) (checkoutURL string, err error)

	CompletePayment(ctx context.Context, token string) (transactionID string, err error)
}

type DefaultService struct {
	userService inboxuser.Service
	// how long does a payment permit the user to use the service?
	paymentValidity time.Duration
	plans           map[inboxuser.Plan]PlanDetails
	processor       Processor
}

func NewService(userService inboxuser.Service, processor Processor, paymentValidity time.Duration) (*DefaultService, error) {
	plans, err := readPlans()
	if err != nil {
		return nil, err
	}

	return &DefaultService{
		userService:     userService,
		paymentValidity: paymentValidity,
		plans:           plans,
		processor:       processor,
	}, nil
}

func (s DefaultService) VerifyPayment(ctx context.Context) (*PaymentInfo, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userService.FindByAddress(ctx, addr)
	if err != nil {
		return nil, err
	}

	var currentPayment inboxuser.Payment
	var validPaymentExists bool
	now := time.Now()
	for _, p := range user.Payments {
		if p.Timestamp.Add(s.paymentValidity).After(now) {
			currentPayment = p
			validPaymentExists = true
			break
		}
	}

	if !validPaymentExists {
		return nil, mailboxsweep.NeedsPaymentError(nil)
	}

	return &PaymentInfo{
		Plan:       currentPayment.Plan,
		ValidUntil: currentPayment.Timestamp.Add(s.paymentValidity),
	}, nil
}

func (s DefaultService) InitiatePayment(ctx context.Context, plan inboxuser.Plan) (checkoutURL string, err error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return "", err
	}

	planDetails, ok := s.plans[plan]
	if !ok {
		err := fmt.Errorf("plan %s not recognized", plan)
		return "", mailboxsweep.NotFoundError(err)
	}

	return s.processor.InitiateCheckout(int64(planDetails.PriceInCents), addr, planDetails.DisplayName, planDetails.Description)
}

func (s DefaultService) CompletePayment(ctx context.Context, token string) (transactionID string, err error) {
	log.Printf("Unimplemented: got completion token %s for stripe\n", token)
	return "", errors.New("UNIMPLEMENTED: STORE PAYMENT DATA")
}
