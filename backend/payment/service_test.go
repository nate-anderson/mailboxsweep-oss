package payment

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPaymentService(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	cartRepo := NewCartRepository(db, table)
	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	userService, err := inboxuser.NewService(userRepo)
	assert.ThatError(t, err).IsNil()

	processor := NewTestProcessor()
	processor.SessionID = uuid.NewString()

	service, err := NewService(userService, processor, cartRepo, time.Hour)
	assert.ThatError(t, err).IsNil()

	testAddr := "money@bags.com"
	_, err = userService.Store(ctx, testAddr)
	assert.ThatError(t, err).IsNil()

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddr, t)

	t.Run("verify payment, unauthenticated", func(t *testing.T) {
		pmt, err := service.VerifyPayment(ctx)
		assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		assert.That(t, pmt).Equals(nil)
	})

	t.Run("verify payment, no payment", func(t *testing.T) {
		pmt, err := service.VerifyPayment(authedCtx)
		assert.ThatError(t, err).Is(mailboxsweep.ErrNeedsPayment)
		assert.That(t, pmt).Equals(nil)
	})

	t.Run("initiate payment, success", func(t *testing.T) {
		checkoutURL, err := service.InitiatePayment(authedCtx, inboxuser.Deluxe)
		assert.ThatError(t, err).IsNil()
		assert.That(t, checkoutURL).Equals(TestCheckoutURL)

		inProgress, err := cartRepo.find(ctx, inboxuser.PaymentMethod("Test"), processor.SessionID)
		assert.ThatError(t, err).IsNil()
		assert.That(t, inProgress.UserAddress).Equals(testAddr)
		assert.That(t, inProgress.Plan).Equals(inboxuser.Deluxe)
		assert.That(t, inProgress.Method).Equals(inboxuser.PaymentMethod("Test"))
		assert.That(t, inProgress.CreatedAt.IsZero()).Equals(false)
		assert.ThatOrdered(t, inProgress.Expires).GreaterThan(0)
	})

	t.Run("verify payment, incomplete", func(t *testing.T) {
		pmt, err := service.VerifyPayment(authedCtx)
		assert.ThatError(t, err).Is(mailboxsweep.ErrNeedsPayment)
		assert.That(t, pmt).Equals(nil)
	})

	t.Run("complete payment, unauthenticated", func(t *testing.T) {
		err := service.CompletePayment(ctx, inboxuser.PaymentMethod("Test"), "12345")
		assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
	})

	t.Run("complete payment, no cart stored", func(t *testing.T) {
		newAddr := "nobody@notsignedup.com"
		_, err := userService.Store(ctx, newAddr)
		assert.ThatError(t, err).IsNil()
		noStoredCtx := mailboxsweep.AuthorizeTestContext(ctx, newAddr, t)
		err = service.CompletePayment(noStoredCtx, inboxuser.PaymentMethod("Test"), "12345")
		assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
	})

	t.Run("complete payment, success", func(t *testing.T) {
		err := cartRepo.store(authedCtx, cart{
			ExternalID:  processor.SessionID,
			Method:      inboxuser.Stripe,
			UserAddress: testAddr,
			Plan:        inboxuser.Deluxe,
			CreatedAt:   time.Now(),
			Status:      open,
			Expires:     time.Now().Add(time.Minute).Unix(),
		})
		assert.ThatError(t, err).IsNil()
		err = service.CompletePayment(authedCtx, inboxuser.PaymentMethod("Test"), processor.SessionID)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("verify payment, success", func(t *testing.T) {
		pmt, err := service.VerifyPayment(authedCtx)
		assert.ThatError(t, err).IsNil()
		assert.That(t, pmt.Plan).Equals(inboxuser.Deluxe)
		assert.That(t, pmt.ValidUntil.IsZero()).Equals(false)
		assert.That(t, pmt.StartedAt.Day()).Equals(time.Now().Day())
		assert.That(t, pmt.StartedAt.Month()).Equals(time.Now().Month())
		assert.That(t, pmt.StartedAt.Year()).Equals(time.Now().Year())
		assert.That(t, pmt.StartedAt.Hour()).Equals(time.Now().Hour())
	})

	t.Run("initiate payment, cannot pay when a payment is active, failure", func(t *testing.T) {
		checkoutURL, err := service.InitiatePayment(authedCtx, inboxuser.Deluxe)
		assert.ThatError(t, err).Is(mailboxsweep.ErrAlreadyPaid)
		assert.That(t, checkoutURL).Equals("")
	})

}
