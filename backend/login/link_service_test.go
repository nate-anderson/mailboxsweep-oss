package login

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLoginService(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	userService, err := inboxuser.NewService(userRepo, logrus.New())
	assert.ThatError(t, err).IsNil()

	sender := &testSender{}

	cert, err := ReadTestCert(t)
	assert.ThatError(t, err).IsNil()

	repo := NewDynamoLinkRepository(db, table)
	service := NewLinkService(repo, userService, sender, cert, time.Minute, 20)

	testAddress := "admin3@example.org"
	testIP := "192.168.0.24"
	user, err := userService.Store(ctx, testAddress)
	assert.ThatError(t, err).IsNil()

	var (
		token     string
		signature string
	)

	t.Run("create login link", func(t *testing.T) {
		t.Run("existing user", func(t *testing.T) {
			t.Run("create", func(t *testing.T) {
				err := service.CreateLoginLink(ctx, *user, testIP)
				assert.ThatError(t, err).IsNil()
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
				token = link.Token
				signature = sender.req.Signature
			})
		})

	})

	t.Run("validate login link", func(t *testing.T) {
		t.Run("invalid signature", func(t *testing.T) {
			sigCopy := signature + "1"
			userUUID, err := service.ValidateLoginLink(ctx, token, sigCopy)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, userUUID).Equals("")
		})

		t.Run("invalid token", func(t *testing.T) {
			userUUID, err := service.ValidateLoginLink(ctx, token+"1", signature)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, userUUID).Equals("")
		})

		t.Run("success", func(t *testing.T) {
			userUUID, err := service.ValidateLoginLink(ctx, token, signature)
			assert.ThatError(t, err).IsNil()
			assert.That(t, userUUID).Equals(user.UUID)
		})
	})
}
