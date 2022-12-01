package inbox

import (
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestIMAPFolder(t *testing.T) {
	testConf, err := mailboxsweep.GetTestConfig()
	if err != nil {
		panic(err)
	}

	provider, err := NewIMAPProvider(
		testConf.IMAPHost,
		testConf.IMAPPort,
	)
	assert.ThatError(t, err).IsNil()

	inbox, err := provider.Login(
		testConf.Address,
		testConf.Password,
	)
	assert.ThatError(t, err).IsNil()

	folder, err := inbox.Folder("INBOX")
	assert.ThatError(t, err).IsNil()

	n := 5

	sender := mailboxsweep.NewTestSMTPSender(testConf)

	for i := 0; i < n; i++ {
		subj := fmt.Sprintf("SUBJ %d", i)
		msg := fmt.Sprintf("MSG %d", i)
		err := sender.Send(testConf.Address, testConf.Address, subj, msg)
		if err != nil {
			t.Error(err)
		}
	}

	t.Run("get messages", func(t *testing.T) {
		mChan := make(chan Message)
		errChan := make(chan error)

		msgs := make([]Message, n)

		go folder.Messages(1, uint32(n), mChan, errChan)

		for {
			select {
			case msg, ok := <-mChan:
				if !ok {
					goto Done
				}
				msgs = append(msgs, msg)
			case err, ok := <-errChan:
				if !ok {
					goto Done
				}
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	Done:
		assert.That(t, len(msgs) >= n).Equals(true)
	})

}
