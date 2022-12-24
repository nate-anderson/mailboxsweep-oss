package inbox

import (
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"sync"
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
		wg := sync.WaitGroup{}

		go folder.Messages(1, uint32(n), mChan, errChan)

		wg.Add(1)
		go func() {
			for {
				select {
				case msg := <-mChan:
					msgs = append(msgs, msg)
				case err := <-errChan:
					wg.Done()
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}()

		wg.Wait()

		assert.That(t, len(msgs) >= n).Equals(true)
	})

}
