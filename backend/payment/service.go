package payment

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"time"
)

type PaymentInfo struct {
	Plan       inboxuser.Plan
	StartedAt  time.Time
	ValidUntil time.Time
}

type Service interface {
	// returns the active user's plan if they have paid, else nil and ErrNeedsPayment
	VerifyPayment(ctx context.Context) (*PaymentInfo, error)
	// returns the checkout redirect URL
	// returns ErrAlreadyPaid if a payment is current
	InitiatePayment(ctx context.Context, plan inboxuser.Plan) (checkoutURL string, err error)
	CompletePayment(ctx context.Context, method inboxuser.PaymentMethod, token string) error
	PlanDetails(code inboxuser.Plan) (*PlanDetails, error)
	Plans(ctx context.Context) (map[inboxuser.Plan]PlanDetails, error)
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
		return nil, mailboxsweep.NeedsPaymentError(errors.New("no current payment"))
	}

	return &PaymentInfo{
		Plan:       currentPayment.Plan,
		StartedAt:  currentPayment.Timestamp,
		ValidUntil: currentPayment.Timestamp.Add(s.paymentValidity),
	}, nil
}

func (c DefaultService) method() (inboxuser.PaymentMethod, error) {
	switch c.processor.(type) {
	case StripeProcessor, *StripeProcessor:
		return inboxuser.Stripe, nil
	case *TestProcessor:
		return inboxuser.PaymentMethod("Test"), nil
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

	_, err = s.VerifyPayment(ctx)
	if !errors.Is(err, mailboxsweep.ErrNeedsPayment) {
		return "", mailboxsweep.ErrAlreadyPaid
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
		Status:      open,
	}
	err = s.cartRepo.store(ctx, c)
	if err != nil {
		return "", err
	}

	return checkoutURL, nil
}

func (s DefaultService) CompletePayment(ctx context.Context, method inboxuser.PaymentMethod, token string) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	session, err := s.processor.FindCheckoutSession(token)
	if err != nil {
		return err
	}

	cart, err := s.cartRepo.find(ctx, method, token)
	if err != nil {
		return err
	}

	if cart.UserAddress != addr {
		return errors.New("cart address mismatch")
	}

	if cart.Status != open {
		return fmt.Errorf("cart not open: status %s", cart.Status)
	}

	pmt := inboxuser.Payment{
		Plan:          cart.Plan,
		AmountInCents: session.TotalInCents,
		Method:        inboxuser.Stripe,
		ExternalID:    session.SessionID,
		Timestamp:     time.Now(),
	}
	if err = s.userService.StorePayment(ctx, pmt); err != nil {
		return err
	}

	if err = s.cartRepo.updateStatus(ctx, cart.Method, cart.ExternalID, complete); err != nil {
		return err
	}

	return err
}

func (s DefaultService) PlanDetails(code inboxuser.Plan) (*PlanDetails, error) {
	plan, ok := s.plans[code]
	if !ok {
		err := fmt.Errorf("plan '%s' not found", code)
		return nil, mailboxsweep.NotFoundError(err)
	}
	return &plan, nil
}

func (s DefaultService) Plans(ctx context.Context) (map[inboxuser.Plan]PlanDetails, error) {
	_, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.plans, nil
}
