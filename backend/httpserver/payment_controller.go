package httpserver

import (
	"encoding/json"
	"errors"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"mailboxsweep/payment"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type PaymentController struct {
	baseController
	service payment.Service
	log     *logrus.Logger
}

func NewPaymentController(service payment.Service, log *logrus.Logger) *PaymentController {
	return &PaymentController{
		baseController: baseController{log},
		log:            log,
		service:        service,
	}
}

func (c PaymentController) GetPaymentStatus(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	status, err := c.service.VerifyPayment(ctx)
	if err != nil {
		if errors.Is(err, mailboxsweep.ErrNeedsPayment) {
			c.respond(w, http.StatusOK, UserPaymentStatus{})
			return
		}
		c.respondError(w, err)
		return
	}

	plan, err := c.service.PlanDetails(status.Plan)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resBody := UserPaymentStatus{
		Plan: Plan{
			AllowedScans: plan.AllowedScans,
			Description:  plan.Description,
			DisplayName:  plan.DisplayName,
			DurationDays: plan.DurationDays,
			Features:     plan.Features,
			Key:          string(status.Plan),
			PriceInCents: plan.PriceInCents,
		},
		ValidUntil:   ptr(status.ValidUntil.Format(time.RFC3339)),
		SubscribedAt: ptr(status.ValidUntil.Format(time.RFC3339)),
	}

	c.respond(w, http.StatusOK, resBody)
}

func (c PaymentController) InitiatePayment(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := InitiatePaymentRequest{}
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	plan, err := inboxuser.ParsePlan(reqBody.Plan)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	checkoutURL, err := c.service.InitiatePayment(ctx, plan)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resBody := InitiatePaymentResponse{
		CheckoutUrl: &checkoutURL,
	}

	c.respond(w, http.StatusOK, resBody)
}

func (c PaymentController) CompletePayment(w http.ResponseWriter, req *http.Request, params CompletePaymentParams) {
	ctx := req.Context()
	err := c.service.CompletePayment(ctx, params.CheckoutSessionId)
	if err != nil {
		c.respondError(w, err)
		return
	}

	msg := "Payment received!"
	resBody := StatusResponse{
		Status:  "OK",
		Message: &msg,
	}
	c.respond(w, http.StatusOK, resBody)
}

func (c PaymentController) ListPlans(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	planMap, err := c.service.Plans(ctx)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resPlans := make([]Plan, 0, len(planMap))
	for key, plan := range planMap {
		resPlans = append(resPlans, Plan{
			AllowedScans: plan.AllowedScans,
			Description:  plan.Description,
			DisplayName:  plan.DisplayName,
			DurationDays: plan.DurationDays,
			Features:     plan.Features,
			Key:          string(key),
			PriceInCents: plan.PriceInCents,
		})
	}

	resBody := ListPlansResponse{
		Plans: resPlans,
	}

	c.respond(w, http.StatusOK, resBody)
}
