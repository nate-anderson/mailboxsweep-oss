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

	addr := "user@mailboxsweep.com"
	userServer := Server{
		Domain:    "mailboxsweep.com",
		Host:      "imap.mailboxsweep.com",
		CreatedBy: &addr,
		Port:      993,
		IsGlobal:  false,
	}

	globalServer := Server{
		Domain:   "domain.com",
		Host:     "imap.domain.com",
		Port:     993,
		IsGlobal: true,
	}
	err := repo.Store(ctx, globalServer)
	assert.ThatError(t, err).IsNil()

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, addr, t)

	t.Run("store", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.Store(context.Background(), userServer)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)

			found, err := service.Find(authedCtx)
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
			err := service.Store(authedCtx, userServer)
			assert.ThatError(t, err).IsNil()
		})
	})

	t.Run("find", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			s, err := service.Find(context.Background())
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, s).Equals(nil)
		})

		t.Run("not found, failure", func(t *testing.T) {
			authedCtx2 := mailboxsweep.AuthorizeAPIContext(ctx, "beans@gooble.com")
			s, err := service.Find(authedCtx2)
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, s).Equals(nil)
		})

		t.Run("success - user server", func(t *testing.T) {
			s, err := service.Find(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, *s.CreatedBy).Equals(addr)
			assert.That(t, s.Domain).Equals(userServer.Domain)
			assert.That(t, s.Host).Equals(userServer.Host)
			assert.That(t, s.Port).Equals(userServer.Port)
			assert.That(t, s.IsGlobal).Equals(false)
		})

		t.Run("success - global server", func(t *testing.T) {
			authedCtx2 := mailboxsweep.AuthorizeAPIContext(ctx, "user@domain.com")
			s, err := service.Find(authedCtx2)
			assert.ThatError(t, err).IsNil()
			assert.That(t, s.Domain).Equals(globalServer.Domain)
			assert.That(t, s.Host).Equals(globalServer.Host)
			assert.That(t, s.Port).Equals(globalServer.Port)
			assert.That(t, s.IsGlobal).Equals(true)
		})
	})
}
