package inbox

import (
	"mailboxsweep/assert"
	"testing"
)

func TestIMAPInbox(t *testing.T) {
	provider, err := NewTestIMAPProvider("localhost", 3143)
	assert.ThatError(t, err).IsNil()

	inbox, err := provider.Login("testuser@localhost", "buddy")
	assert.ThatError(t, err).IsNil()

	var folders []string

	t.Run("get folders", func(t *testing.T) {
		var err error
		folders, err = inbox.Folders()
		assert.ThatError(t, err).IsNil()

		assert.ThatSlice(t, folders).HasLength(1)
		assert.That(t, folders[0]).Equals("INBOX")
	})

	t.Run("get named folder", func(t *testing.T) {
		folder, err := inbox.Folder("INBOX")
		assert.ThatError(t, err).IsNil()

		assert.That(t, folder.Count()).Equals(0)
	})

	t.Run("get validity keys", func(t *testing.T) {
		keys, err := inbox.ValidityKeys()
		assert.ThatError(t, err).IsNil()
		assert.ThatOrdered(t, len(keys)).GreaterThanOrEqual(1)
		for key, value := range keys {
			assert.That(t, value).NotEqual(0)
			assert.ThatSlice(t, folders).Contains(key)
		}
	})
}
