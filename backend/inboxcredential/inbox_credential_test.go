package inboxcredential

import (
	"crypto/aes"
	"crypto/cipher"
	"mailboxsweep/assert"
	"testing"
)

func TestInboxCredential(t *testing.T) {
	testAddr := "admin@example.com"
	password := "test_password_#_123"
	cipherKey := []byte("123_abc_456_defg")

	block, err := aes.NewCipher(cipherKey)
	assert.ThatError(t, err).IsNil()

	encryptor, err := cipher.NewGCM(block)
	assert.ThatError(t, err).IsNil()

	t.Run("IMAP inbox credential", func(t *testing.T) {
		var i *IMAPInboxCredential
		t.Run("encrypt", func(t *testing.T) {
			var err error
			i, err = NewIMAPInboxCredential(testAddr, password)
			assert.ThatError(t, err).IsNil()

			assert.That(t, i.EmailAddress).Equals(testAddr)
			assert.That(t, len(i.Password) > 0).Equals(true)

			err = i.encrypt(encryptor, encryptor.NonceSize())
			assert.ThatError(t, err).IsNil()
			assert.That(t, i.Nonce).NotEqual("")
			assert.That(t, i.Password).NotEqual(password)
			assert.That(t, i.Address()).Equals(testAddr)
		})

		t.Run("decrypt", func(t *testing.T) {
			err := i.decrypt(encryptor)
			assert.ThatError(t, err).IsNil()
			assert.That(t, i.Password).Equals(password)
		})
	})

}
