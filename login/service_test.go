package login

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"mailboxsweep/session"
	"testing"
	"time"
)

func TestService(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	userService := inboxuser.NewService(userRepo)

	sender := &testSender{}

	sessionRepo := session.NewDynamoSessionRepository(db, table)
	sessionService := session.NewService(sessionRepo, 10, time.Minute)

	repo := NewDynamoLinkRepository(db, table)
	service := NewService(repo, userService, sessionService, sender, time.Minute, 20)

	testAddress := "admin3@example.org"
	testIP := "192.168.0.24"
	user, err := userService.Store(ctx, testAddress)
	assert.ThatError(t, err).IsNil()

	t.Run("create login link", func(t *testing.T) {
		t.Run("existing user", func(t *testing.T) {
			t.SkipNow()
			t.Run("create", func(t *testing.T) {
				existing, err := service.CreateLoginLink(ctx, testAddress, testIP)
				assert.ThatError(t, err).IsNil()
				assert.That(t, existing).Equals(true)
				assert.That(t, sender.req.Address).Equals(testAddress)
			})

			t.Run("retrieve", func(t *testing.T) {
				link, err := repo.FindByUserUUID(ctx, user.UUID)
				assert.ThatError(t, err).IsNil()
				assert.That(t, link.UserUUID).Equals(user.UUID)
				assert.ThatOrdered(t, link.Expires).GreaterThan(0)
				assert.That(t, link.Created.IsZero()).Equals(false)
				assert.That(t, link.InitiatedByIP).Equals(testIP)
				assert.That(t, sender.req.Token).Equals(link.Token)
			})
		})

		t.Run("new user", func(t *testing.T) {
			testNewAddress := "bambam@flintstone.org"
			testNewIP := "192.168.0.22"
			t.Run("create", func(t *testing.T) {
				existing, err := service.CreateLoginLink(ctx, testNewAddress, testNewIP)
				assert.ThatError(t, err).IsNil()
				assert.That(t, existing).Equals(false)
				assert.That(t, sender.req.Address).Equals(testNewAddress)
			})

			t.Run("retrieve", func(t *testing.T) {
				link, err := repo.Find(ctx, sender.req.Token)
				assert.ThatError(t, err).IsNil()

				user, err := userService.FindByAddress(ctx, testNewAddress)
				assert.ThatError(t, err).IsNil()

				assert.That(t, link.UserUUID).Equals(user.UUID)
				assert.ThatOrdered(t, link.Expires).GreaterThan(0)
				assert.That(t, link.Created.IsZero()).Equals(false)
				assert.That(t, link.InitiatedByIP).Equals(testNewIP)
				assert.That(t, sender.req.Token).Equals(link.Token)
			})
		})
	})
}
