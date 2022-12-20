package inboxuser

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
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
	})
}
