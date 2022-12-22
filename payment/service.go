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

	CompletePayment(ctx context.Context, token string) error
}

type DefaultService struct {
	userService inboxuser.Service
	cartRepo    *CartRepository
	// how long does a payment permit the user to use the service?
	paymentValidity time.Duration
	plans           map[inboxuser.Plan]PlanDetails
	processor       Processor
}

func NewService(userService inboxuser.Service, processor Processor, cartRepo *CartRepository, paymentValidity time.Duration) (*DefaultService, error) {
	if paymentValidity == 0 {
		return nil, errors.New("payment validity period must be greater than 0")
	}

	plans, err := readPlans()
	if err != nil {
		return nil, err
	}

	return &DefaultService{
		userService:     userService,
		paymentValidity: paymentValidity,
		cartRepo:        cartRepo,
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

func (c DefaultService) method() (inboxuser.PaymentMethod, error) {
	switch c.processor.(type) {
	case StripeProcessor, *StripeProcessor:
		return inboxuser.Stripe, nil
	default:
		return "", fmt.Errorf("internal: the configured processor is of an unknown type")
	}
}

const storeInProgressCartsHours = 1

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

	checkoutURL, sessionID, err := s.processor.InitiateCheckout(int64(planDetails.PriceInCents), addr, planDetails.DisplayName, planDetails.Description)
	if err != nil {
		return "", err
	}

	method, err := s.method()
	if err != nil {
		return "", err
	}

	c := cart{
		ExternalID:  sessionID,
		Method:      method,
		UserAddress: addr,
		Plan:        plan,
		CreatedAt:   time.Now(),
		Expires:     time.Now().Add(storeInProgressCartsHours * time.Hour).Unix(),
	}
	err = s.cartRepo.store(ctx, c)
	if err != nil {
		return "", err
	}

	return checkoutURL, nil
}

func (s DefaultService) CompletePayment(ctx context.Context, token string) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	complete, err := s.processor.FindCheckoutSession(token)
	if err != nil {
		return err
	}

	cart, err := s.cartRepo.find(ctx, inboxuser.Stripe, token)
	if err != nil {
		return mailboxsweep.NotFoundError(err)
	}

	if cart.UserAddress != addr {
		log.Printf("want %s, got %s", cart.UserAddress, addr)
		return errors.New("cart address mismatch")
	}

	pmt := inboxuser.Payment{
		Plan:          cart.Plan,
		AmountInCents: complete.TotalInCents,
		Method:        inboxuser.Stripe,
		ExternalID:    complete.SessionID,
		Timestamp:     time.Now(),
	}
	err = s.userService.StorePayment(ctx, pmt)
	log.Println("payment stored")
	return err
}
