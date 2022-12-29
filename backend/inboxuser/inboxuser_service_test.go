package inboxuser

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"
)

func TestInboxUserService(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewDynamoInboxUserRepository(db, table)
	service, err := NewService(repo)
	assert.ThatError(t, err).IsNil()
	testAddr := "beans@bacon.net"

	t.Run("store user", func(t *testing.T) {
		ctx := context.Background()
		user, err := service.Store(ctx, testAddr)
		assert.ThatError(t, err).IsNil()
		assert.That(t, user.Address).Equals(testAddr)
		assert.That(t, user.UUID).NotEqual("")
		assert.That(t, user.NameFirst).Equals("")
		assert.That(t, user.NameLast).Equals("")
	})

	authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, testAddr)
	t.Run("update user name", func(t *testing.T) {
		err := service.SetName(authedCtx, "First", "Last")
		assert.ThatError(t, err).IsNil()

		user, err := service.FindByAddress(ctx, testAddr)
		assert.ThatError(t, err).IsNil()
		assert.That(t, user.Address).Equals(testAddr)
		assert.That(t, user.NameFirst).Equals("First")
		assert.That(t, user.NameLast).Equals("Last")
	})

	t.Run("user agreement", func(t *testing.T) {
		agreements := []Agreement{PersonalInbox, NecessaryCookies}
		t.Run("consent", func(t *testing.T) {
			err := service.Consent(authedCtx, agreements)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("list agreements", func(t *testing.T) {
			userAgreements, err := service.ListUserAgreements(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(userAgreements)).Equals(len(allAgreements))
			for _, ua := range userAgreements {
				if contains(agreements, ua.Agreement) {
					assert.That(t, ua.Agreed).Equals(true)
					assert.That(t, ua.AgreedAt.IsZero()).Equals(false)
				} else {
					assert.That(t, ua.Agreed).Equals(false)
					assert.That(t, ua.AgreedAt.IsZero()).Equals(true)
				}
			}
		})

		t.Run("get agreement", func(t *testing.T) {
			t.Run("unauthorized, error", func(t *testing.T) {
				ua, err := service.GetAgreement(ctx, PersonalInbox)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
				assert.That(t, ua).Equals(nil)
			})

			t.Run("authorized, agreeement not found", func(t *testing.T) {
				ua, err := service.GetAgreement(authedCtx, Agreement("BS"))
				assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
				assert.That(t, ua).Equals(nil)
			})

			t.Run("authorized, success, agreed", func(t *testing.T) {
				ua, err := service.GetAgreement(authedCtx, PersonalInbox)
				assert.ThatError(t, err).IsNil()
				assert.That(t, ua.Agreed).Equals(true)
				assert.That(t, ua.AgreedAt.IsZero()).Equals(false)
				assert.That(t, ua.Agreement).Equals(PersonalInbox)
				assert.That(t, ua.Description).Equals(service.agreements[PersonalInbox].Body)
				assert.That(t, ua.Required).Equals(true)
			})
		})
	})

	t.Run("store payment", func(t *testing.T) {
		testPayment := Payment{
			Plan:          Basic,
			AmountInCents: 1000,
			Method:        Stripe,
			ExternalID:    "12345",
			Timestamp:     time.Now(),
		}
		err := service.StorePayment(authedCtx, testPayment)
		assert.ThatError(t, err).IsNil()

		user, err := service.FindByAddress(authedCtx, testAddr)
		assert.ThatError(t, err).IsNil()
		assert.That(t, len(user.Payments)).Equals(1)
		p := user.Payments[0]
		assert.That(t, p.AmountInCents).Equals(testPayment.AmountInCents)
		assert.That(t, p.Plan).Equals(testPayment.Plan)
		assert.That(t, p.Method).Equals(testPayment.Method)
		assert.That(t, p.ExternalID).Equals(testPayment.ExternalID)
		assert.That(t, p.Timestamp.IsZero()).Equals(false)
	})
}
