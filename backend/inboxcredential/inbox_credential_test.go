package inboxcredential

import (
	"mailboxsweep/assert"
	"testing"
)

func TestInboxCredential(t *testing.T) {
	testAddr := "admin@example.com"
	password := "test_password_#_123"
	cipherKey := []byte("123_abc_456_defg")

	var i *InboxCredential
	t.Run("encrypt", func(t *testing.T) {
		var err error
		i, err = NewInboxCredential(testAddr, password, cipherKey)
		assert.ThatError(t, err).IsNil()

		assert.That(t, i.Address).Equals(testAddr)
		assert.That(t, i.Nonce).NotEqual(nil)
		assert.That(t, len(i.Password) > 0).Equals(true)
	})

	t.Run("decrypt", func(t *testing.T) {
		pass, err := i.decryptPassword(cipherKey)
		assert.ThatError(t, err).IsNil()
		assert.That(t, pass).Equals(password)
	})

}
