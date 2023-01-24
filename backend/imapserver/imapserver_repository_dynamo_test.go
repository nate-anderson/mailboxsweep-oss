package imapserver

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestDynamoIMAPServerRepository(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewIMAPServerRepository(db, table)

	server := Server{
		Domain: "gmail.com",
		Host:   "imap.gmail.com",
		Port:   993,
	}

	t.Run("store", func(t *testing.T) {
		err := repo.Store(context.Background(), server)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("retrieve", func(t *testing.T) {
		s, err := repo.FindByDomain(context.Background(), server.Domain)
		assert.ThatError(t, err).IsNil()
		assert.That(t, *s).Equals(server)
	})

	t.Run("update", func(t *testing.T) {
		server.Port = 994
		err := repo.Store(context.Background(), server)
		assert.ThatError(t, err).IsNil()
		s, err := repo.FindByDomain(context.Background(), server.Domain)
		assert.ThatError(t, err).IsNil()
		assert.That(t, *s).Equals(server)
	})
}
