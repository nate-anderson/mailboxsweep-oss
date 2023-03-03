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

	t.Run("global server connection info", func(t *testing.T) {
		globalServer := Server{
			Domain:   "gmail.com",
			Host:     "imap.gmail.com",
			Port:     993,
			IsGlobal: true,
		}

		t.Run("store", func(t *testing.T) {
			err := repo.Store(context.Background(), globalServer)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("retrieve", func(t *testing.T) {
			s, err := repo.FindByDomain(context.Background(), globalServer.Domain)
			assert.ThatError(t, err).IsNil()
			assert.That(t, *s).Equals(globalServer)
		})

		t.Run("update", func(t *testing.T) {
			globalServer.Port = 994
			err := repo.Store(context.Background(), globalServer)
			assert.ThatError(t, err).IsNil()
			s, err := repo.FindByDomain(context.Background(), globalServer.Domain)
			assert.ThatError(t, err).IsNil()
			assert.That(t, *s).Equals(globalServer)
		})
	})

	t.Run("user-scoped server info", func(t *testing.T) {
		addr := "me@mailboxsweep.com"
		userServer := Server{
			Domain:    "mailboxsweep.com",
			CreatedBy: &addr,
			IsGlobal:  false,
			Host:      "imap.mailboxsweep.com",
			Port:      93,
		}

		t.Run("store", func(t *testing.T) {
			err := repo.Store(ctx, userServer)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("find by user address should succeed", func(t *testing.T) {
			s, err := repo.FindByUserAddress(ctx, addr)
			assert.ThatError(t, err).IsNil()

			assert.That(t, *s.CreatedBy).Equals(addr)
			assert.That(t, s.Domain).Equals(userServer.Domain)
			assert.That(t, s.Host).Equals(userServer.Host)
			assert.That(t, s.Port).Equals(userServer.Port)
			assert.That(t, s.IsGlobal).Equals(false)
		})
	})

}
