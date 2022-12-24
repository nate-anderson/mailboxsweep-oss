package payment

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"testing"
	"time"
)

func TestCartRepository(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewCartRepository(db, table)

	testAddr := "testuser@example.com"
	method := inboxuser.Stripe
	testCart := cart{
		ExternalID:  "12345",
		Method:      method,
		UserAddress: testAddr,
		Plan:        inboxuser.Basic,
		CreatedAt:   time.Now(),
		Status:      open,
		Expires:     time.Now().Add(time.Minute).Unix(),
	}

	t.Run("store cart", func(t *testing.T) {

		err := repo.store(ctx, testCart)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("find cart", func(t *testing.T) {
		c, err := repo.find(ctx, method, testCart.ExternalID)
		assert.ThatError(t, err).IsNil()

		assert.That(t, c.ExternalID).Equals(testCart.ExternalID)
		assert.That(t, c.Expires).Equals(testCart.Expires)
		assert.That(t, c.UserAddress).Equals(testCart.UserAddress)
		assert.That(t, c.Method).Equals(method)
		assert.That(t, c.Plan).Equals(testCart.Plan)
		assert.That(t, c.CreatedAt.IsZero()).Equals(false)
		assert.That(t, c.Status).Equals(open)
	})

	t.Run("update cart status", func(t *testing.T) {
		err := repo.updateStatus(ctx, method, testCart.ExternalID, complete)
		assert.ThatError(t, err).IsNil()

		retrieved, err := repo.find(ctx, method, testCart.ExternalID)
		assert.ThatError(t, err).IsNil()
		assert.That(t, retrieved.ExternalID).Equals(testCart.ExternalID)
		assert.That(t, retrieved.Expires).Equals(testCart.Expires)
		assert.That(t, retrieved.UserAddress).Equals(testCart.UserAddress)
		assert.That(t, retrieved.Method).Equals(method)
		assert.That(t, retrieved.Plan).Equals(testCart.Plan)
		assert.That(t, retrieved.CreatedAt.IsZero()).Equals(false)
		assert.That(t, retrieved.Status).Equals(complete)
	})
}
