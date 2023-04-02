package inboxcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestDynamoInboxCredentialRepository(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoInboxCredentialRepository(cl, table)

	address := "admin2@example3.com"
	pass := "test_pass_bingo_bongo_%"
	cipherKey := []byte("987_zxy_456_defg")

	block, err := aes.NewCipher(cipherKey)
	assert.ThatError(t, err).IsNil()

	encryptor, err := cipher.NewGCM(block)
	assert.ThatError(t, err).IsNil()

	cred, err := NewIMAPInboxCredential(address, pass)
	assert.ThatError(t, err).IsNil()

	err = cred.encrypt(encryptor, encryptor.NonceSize())
	assert.ThatError(t, err).IsNil()

	t.Run("upsert credential (insert)", func(t *testing.T) {
		err := repo.Upsert(ctx, cred)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("find credential", func(t *testing.T) {
		t.Run("doesn't exist", func(t *testing.T) {
			retrieved, err := repo.FindByAddress(ctx, "nobody@me.com")
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, retrieved == nil).Equals(true)
		})

		t.Run("exists", func(t *testing.T) {
			retrieved, err := repo.FindByAddress(ctx, address)
			assert.ThatError(t, err).IsNil()
			imapCred, ok := retrieved.(*IMAPInboxCredential)
			assert.That(t, ok).Equals(true)
			assert.That(t, imapCred.EmailAddress).Equals(cred.EmailAddress)
			assert.That(t, imapCred.Nonce).Equals(cred.Nonce)
			assert.That(t, imapCred.Password).NotEqual("")

			err = imapCred.decrypt(encryptor)
			assert.ThatError(t, err).IsNil()
			assert.That(t, imapCred.Password).Equals(pass)
		})
	})

	t.Run("upsert (update)", func(t *testing.T) {
		pass2 := "blunko"
		cred2, err := NewIMAPInboxCredential(address, pass2)
		assert.ThatError(t, err).IsNil()
		err = cred2.encrypt(encryptor, encryptor.NonceSize())
		assert.ThatError(t, err).IsNil()

		err = repo.Upsert(ctx, cred2)
		assert.ThatError(t, err).IsNil()

		retrieved, err := repo.FindByAddress(ctx, address)
		assert.ThatError(t, err).IsNil()
		assert.That(t, retrieved.Address()).Equals(cred2.EmailAddress)

		imapCred, ok := retrieved.(*IMAPInboxCredential)
		assert.That(t, ok).Equals(true)
		assert.That(t, imapCred.Nonce).Equals(cred2.Nonce)
		assert.That(t, imapCred.Password).NotEqual("")

		err = imapCred.decrypt(encryptor)
		assert.ThatError(t, err).IsNil()
		assert.That(t, imapCred.Password).Equals(pass2)
	})

	t.Run("delete", func(t *testing.T) {
		err := repo.DeleteByAddress(ctx, address)
		assert.ThatError(t, err).IsNil()

		retrieved, err := repo.FindByAddress(ctx, address)
		assert.ThatError(t, err).IsNotNil()
		assert.That(t, retrieved == nil).Equals(true)
	})
}
