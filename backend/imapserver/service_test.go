package imapserver

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestIMAPServerService(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewIMAPServerRepository(db, table)
	service := NewService(repo)

	server := Server{
		Domain: "gmail.com",
		Host:   "imap.gmail.com",
		Port:   993,
	}

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, "user@domain.com", t)

	t.Run("store", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.Store(context.Background(), server)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)

			found, err := service.FindByDomain(authedCtx, server.Domain)
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, found).Equals(nil)
		})

		t.Run("invalid domain, failure", func(t *testing.T) {
			badServer := Server{
				Domain: "yagoo",
				Host:   "yagoo.com",
				Port:   993,
			}
			err := service.Store(authedCtx, badServer)
			assert.ThatError(t, err).Is(ErrInvalidDomain)
		})

		t.Run("invalid host, failure", func(t *testing.T) {
			badServer := Server{
				Domain: "yagoo.com",
				Host:   "yagoo",
				Port:   993,
			}
			err := service.Store(authedCtx, badServer)
			assert.ThatError(t, err).Is(ErrInvalidHost)
		})

		t.Run("success", func(t *testing.T) {
			err := service.Store(authedCtx, server)
			assert.ThatError(t, err).IsNil()
		})
	})

	t.Run("find by domain", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			s, err := service.FindByDomain(context.Background(), server.Domain)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, s).Equals(nil)
		})

		t.Run("not found, failure", func(t *testing.T) {
			s, err := service.FindByDomain(authedCtx, "gooble.com")
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, s).Equals(nil)
		})

		t.Run("success", func(t *testing.T) {
			s, err := service.FindByDomain(authedCtx, server.Domain)
			assert.ThatError(t, err).IsNil()
			assert.That(t, *s).Equals(server)
		})
	})
}
