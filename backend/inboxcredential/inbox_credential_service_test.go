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

	repo := NewDynamoInboxCredentialRepository(cl, table)
	cipherKey := []byte("655468576D5A7134")
	service := NewService(repo, cipherKey)

	testAddr := "secure_friend@example.com"
	testPassword := "great_password_123"
	authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, testAddr)

	t.Run("store credential", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			addr, err := service.StoreCredential(ctx, testPassword)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, addr).Equals("")
		})

		t.Run("success", func(t *testing.T) {
			addr, err := service.StoreCredential(authedCtx, testPassword)
			assert.ThatError(t, err).IsNil()
			assert.That(t, addr).Equals(testAddr)
		})
	})

	t.Run("retrieve credential", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			ret, err := service.FindCredential(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, ret).Equals(nil)
		})

		t.Run("success", func(t *testing.T) {
			ret, err := service.FindCredential(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, ret.Password).NotEqual("")
			t.Run("password is encrypted", func(t *testing.T) {
				assert.ThatString(t, ret.Password).NotContains(testPassword)
			})

			t.Run("decrypt password", func(t *testing.T) {
				password, err := service.DecryptCredential(ctx, ret)
				assert.ThatError(t, err).IsNil()
				assert.That(t, password).Equals(testPassword)
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
				assert.That(t, ret).Equals(nil)
			})
		})
	})
}
