package inboxcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"
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

	t.Run("IMAP credential", func(t *testing.T) {
		cred := NewIMAPInboxCredential(address, pass)

		err = cred.encrypt(encryptor)
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
			cred2 := NewIMAPInboxCredential(address, pass2)

			err = cred2.encrypt(encryptor)
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
	})

	t.Run("Gmail credential", func(t *testing.T) {
		var (
			address      = "address3@mailboxsweep.com"
			apiToken     = "api_token_123"
			refreshToken = "refresh_token_456"
			expiry       = time.Now().Add(time.Hour)
		)
		cred := NewGmailInboxCredential(address, apiToken, refreshToken, expiry)

		err = cred.encrypt(encryptor)
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
				gmailCred, ok := retrieved.(*GmailInboxCredential)
				assert.That(t, ok).Equals(true)
				assert.That(t, gmailCred.EmailAddress).Equals(cred.EmailAddress)
				assert.That(t, gmailCred.Nonce).Equals(cred.Nonce)
				assert.That(t, gmailCred.APIToken).NotEqual("")
				assert.That(t, gmailCred.RefreshToken).NotEqual("")
				assert.ThatOrdered(t, gmailCred.ExpiresAt).GreaterThan(0)
				assert.That(t, gmailCred.Encrypted).Equals(true)

				err = gmailCred.decrypt(encryptor)
				assert.ThatError(t, err).IsNil()
				assert.That(t, gmailCred.APIToken).Equals(apiToken)
				assert.That(t, gmailCred.RefreshToken).Equals(refreshToken)
			})
		})

		t.Run("upsert (update)", func(t *testing.T) {
			apiToken2 := "blunko2"
			cred2 := NewGmailInboxCredential(address, apiToken2, refreshToken, expiry)

			err = cred2.encrypt(encryptor)
			assert.ThatError(t, err).IsNil()

			err = repo.Upsert(ctx, cred2)
			assert.ThatError(t, err).IsNil()

			retrieved, err := repo.FindByAddress(ctx, address)
			assert.ThatError(t, err).IsNil()
			assert.That(t, retrieved.Address()).Equals(cred2.EmailAddress)

			gmailCred, ok := retrieved.(*GmailInboxCredential)
			assert.That(t, ok).Equals(true)
			assert.That(t, gmailCred.EmailAddress).Equals(cred.EmailAddress)
			assert.That(t, gmailCred.Nonce).Equals(cred2.Nonce)
			assert.That(t, gmailCred.APIToken).NotEqual("")
			assert.That(t, gmailCred.RefreshToken).NotEqual("")
			assert.ThatOrdered(t, gmailCred.ExpiresAt).GreaterThan(0)
			assert.That(t, gmailCred.Encrypted).Equals(true)

			err = gmailCred.decrypt(encryptor)
			assert.ThatError(t, err).IsNil()
			assert.That(t, gmailCred.APIToken).Equals(apiToken2)
			assert.That(t, gmailCred.RefreshToken).Equals(refreshToken)
		})

		t.Run("delete", func(t *testing.T) {
			err := repo.DeleteByAddress(ctx, address)
			assert.ThatError(t, err).IsNil()

			retrieved, err := repo.FindByAddress(ctx, address)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, retrieved == nil).Equals(true)
		})
	})

}
