package inbox

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxcredential"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestIMAPInbox(t *testing.T) {
	testConf, err := mailboxsweep.GetTestConfig()
	assert.ThatError(t, err).IsNil()
	ctx := context.Background()
	provider, err := NewIMAPProvider(ctx, testConf.IMAPHost, testConf.IMAPPort, testConf.Address, nil, nil, logrus.New())
	assert.ThatError(t, err).IsNil()

	imapCred := inboxcredential.NewIMAPInboxCredential(testConf.Address, testConf.Password)

	inbox, err := provider.Login(ctx, testConf.Address, imapCred)
	assert.ThatError(t, err).IsNil()

	var folders []string

	t.Run("get folders", func(t *testing.T) {
		var err error
		folders, err = inbox.Folders()
		assert.ThatError(t, err).IsNil()

		assert.ThatOrdered(t, len(folders)).GreaterThanOrEqual(1)
		assert.ThatSlice(t, folders).Contains("INBOX")
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
