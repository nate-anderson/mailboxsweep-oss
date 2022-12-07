package login

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDynamoLoginRepository(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewDynamoLinkRepository(db, table)

	link := Link{
		UserUUID:      uuid.NewString(),
		Token:         "test_token_1234",
		Expires:       time.Now().Add(time.Minute).Unix(),
		Created:       time.Now(),
		InitiatedByIP: "192.168.0.24",
	}

	t.Run("create login link", func(t *testing.T) {
		err := repo.Store(ctx, link)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("fetch login link", func(t *testing.T) {
		retrieved, err := repo.Find(ctx, link.Token)
		assert.ThatError(t, err).IsNil()
		assert.That(t, retrieved.Created.Equal(link.Created)).Equals(true)
		assert.That(t, retrieved.InitiatedByIP).Equals(link.InitiatedByIP)
		assert.That(t, retrieved.Token).Equals(link.Token)
		assert.That(t, retrieved.UserUUID).Equals(link.UserUUID)
		assert.That(t, retrieved.Expires).Equals(link.Expires)
	})

	t.Run("invalidate login links by address", func(t *testing.T) {

	})
}
