package mailboxsweep

import (
	"context"
	"mailboxsweep/assert"
	"testing"
)

func TestDynamoInboxCredentialRepository(t *testing.T) {
	cl := SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := setupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoInboxCredentialRepository(cl, table)

	address := "admin2@example3.com"
	pass := "test_pass_bingo_bongo_%"
	cipherKey := []byte("987_zxy_456_defg")

	cred, err := NewInboxCredential(address, pass, cipherKey)
	assert.ThatError(t, err).IsNil()

	t.Run("upsert credential (insert)", func(t *testing.T) {
		err := repo.Upsert(ctx, *cred)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("find credential", func(t *testing.T) {
		t.Run("doesn't exist", func(t *testing.T) {
			retrieved, err := repo.FindByAddress(ctx, "nobody@me.com")
			assert.ThatError(t, err).Is(ErrNotFound)
			assert.That(t, retrieved).Equals(nil)
		})

		t.Run("exists", func(t *testing.T) {
			retrieved, err := repo.FindByAddress(ctx, address)
			assert.ThatError(t, err).IsNil()
			assert.That(t, retrieved.Address).Equals(cred.Address)
			assert.That(t, *retrieved.Nonce).Equals(*cred.Nonce)
			assert.That(t, retrieved.Password).NotEqual("")

			decrypted, err := retrieved.DecryptPassword(cipherKey)
			assert.ThatError(t, err).IsNil()
			assert.That(t, decrypted).Equals(pass)
		})
	})

	t.Run("upsert (update)", func(t *testing.T) {
		pass2 := "blunko"
		cred2, err := NewInboxCredential(address, pass2, cipherKey)
		assert.ThatError(t, err).IsNil()
		err = repo.Upsert(ctx, *cred2)
		assert.ThatError(t, err).IsNil()

		retrieved, err := repo.FindByAddress(ctx, address)
		assert.ThatError(t, err).IsNil()
		assert.That(t, retrieved.Address).Equals(cred2.Address)
		assert.That(t, *retrieved.Nonce).Equals(*cred2.Nonce)
		assert.That(t, retrieved.Password).NotEqual("")

		decrypted, err := retrieved.DecryptPassword(cipherKey)
		assert.ThatError(t, err).IsNil()
		assert.That(t, decrypted).Equals(pass2)
	})

	t.Run("delete", func(t *testing.T) {
		err := repo.DeleteByAddress(ctx, address)
		assert.ThatError(t, err).IsNil()

		retrieved, err := repo.FindByAddress(ctx, address)
		assert.ThatError(t, err).IsNotNil()
		assert.That(t, retrieved).Equals(nil)
	})
}
