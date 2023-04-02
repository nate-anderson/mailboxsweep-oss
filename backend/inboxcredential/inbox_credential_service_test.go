package inboxcredential

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestInboxCredentialService(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	t.Run("IMAP credentials", func(t *testing.T) {
		repo := NewDynamoInboxCredentialRepository(cl, table)
		cipherKey := []byte("655468576D5A7134")
		service, err := NewService(repo, cipherKey)
		assert.ThatError(t, err).IsNil()

		testAddr := "secure_friend@example.com"
		testPassword := "great_password_123"
		authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, testAddr)

		t.Run("store credential", func(t *testing.T) {
			cred := NewIMAPInboxCredential(testAddr, testPassword)

			t.Run("unauthenticated, failure", func(t *testing.T) {
				err := service.StoreCredential(ctx, cred)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			})

			t.Run("success", func(t *testing.T) {
				err := service.StoreCredential(authedCtx, cred)
				assert.ThatError(t, err).IsNil()
			})
		})

		t.Run("retrieve credential", func(t *testing.T) {
			t.Run("unauthenticated, failure", func(t *testing.T) {
				ret, err := service.FindCredential(ctx)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
				assert.That(t, ret == nil).Equals(true)
			})

			t.Run("success", func(t *testing.T) {
				ret, err := service.FindCredential(authedCtx)
				assert.ThatError(t, err).IsNil()

				imapCred, ok := ret.(*IMAPInboxCredential)
				assert.That(t, ok).Equals(true)

				assert.That(t, imapCred.Password).NotEqual("")
				t.Run("password is encrypted", func(t *testing.T) {
					assert.ThatString(t, imapCred.Password).NotContains(testPassword)
				})

				t.Run("decrypt password", func(t *testing.T) {
					decrypted, err := service.DecryptCredential(ctx, ret)
					imapCred, ok := decrypted.(*IMAPInboxCredential)
					assert.That(t, ok).Equals(true)
					assert.ThatError(t, err).IsNil()
					assert.That(t, imapCred.Password).Equals(testPassword)
				})
			})
		})

		t.Run("delete credential", func(t *testing.T) {
			t.Run("unauthenticated, failure", func(t *testing.T) {
				err := service.DeleteCredential(ctx)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			})

			t.Run("success", func(t *testing.T) {
				err := service.DeleteCredential(authedCtx)
				assert.ThatError(t, err).IsNil()

				t.Run("record is not present in DB", func(t *testing.T) {
					ret, err := service.FindCredential(authedCtx)
					assert.ThatError(t, err).Is(mailboxsweep.ErrNoStoredCredential)
					assert.That(t, ret == nil).Equals(true)
				})
			})
		})
	})

}
