package login

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"
)

func TestGmailTokenRepository(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewDynamoGmailTokenRepository(db, table)

	testAddr := "user@example.dos.com"

	t.Run("store", func(t *testing.T) {
		token := GmailLoginToken{
			Address: testAddr,
			Token:   "a_token",
			Cookie:  "a_cookie",
			Expires: time.Now().Add(time.Minute).Unix(),
		}

		err := repo.Store(ctx, token)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			token, err := repo.Get(ctx, "a_token")
			assert.ThatError(t, err).IsNil()
			assert.That(t, token.Token).Equals("a_token")
			assert.That(t, token.Address).Equals(testAddr)
			assert.That(t, token.Expires).NotEqual(0)
			assert.That(t, token.Cookie).Equals("a_cookie")
		})

		t.Run("not found", func(t *testing.T) {
			token, err := repo.Get(ctx, "bs")
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, token).Equals(nil)
		})
	})
}
