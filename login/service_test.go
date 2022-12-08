package login

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
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

	repo := NewDynamoLinkRepository(db, table)
	service := NewService(repo, userService, time.Minute, 20)

	testAddress := "admin3@example.org"
	testIP := "192.168.0.24"
	user, err := userService.Store(ctx, testAddress)
	assert.ThatError(t, err).IsNil()

	t.Run("create login link", func(t *testing.T) {
		t.Run("store", func(t *testing.T) {
			err := service.CreateLoginLink(ctx, testAddress, testIP)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("retrieve", func(t *testing.T) {
			link, err := repo.FindByUserUUID(ctx, user.UUID)
			assert.ThatError(t, err).IsNil()
			assert.That(t, link.UserUUID).Equals(user.UUID)
			assert.ThatOrdered(t, link.Expires).GreaterThan(0)
			assert.That(t, link.Created.IsZero()).Equals(false)
			assert.That(t, link.InitiatedByIP).Equals(testIP)
		})
	})
}
