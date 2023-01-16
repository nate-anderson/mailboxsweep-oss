package inbox

import (
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"strconv"
	"sync"
	"testing"
)

func TestIMAPFolder(t *testing.T) {
	testConf, err := mailboxsweep.GetTestConfig()
	if err != nil {
		panic(err)
	}

	sender := mailboxsweep.NewTestSMTPSender(testConf)

	n := 5
	for i := 0; i < n; i++ {
		subj := fmt.Sprintf("SUBJ %d", i)
		msg := fmt.Sprintf("MSG %d", i)
		err := sender.Send(testConf.Address, testConf.Address, subj, msg)
		if err != nil {
			t.Error(err)
		}
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

	// time.Sleep(time.Second * 60)

	msgs := make([]Message, 0, n)

	t.Run("get messages", func(t *testing.T) {
		mChan := make(chan Message)
		errChan := make(chan error)

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
						panic(err)
					}
				}
			}
		}()

		wg.Wait()

		assert.ThatSlice(t, msgs).HasLength(n)
	})

	msgIDs := make([]string, 0, n)
	for _, m := range msgs {
		msgIDs = append(msgIDs, strconv.FormatUint(uint64(m.UID), 10))
	}

	t.Run("mark messages read", func(t *testing.T) {
		assert.That(t, len(msgIDs)).NotEqual(0)
		err := folder.MarkRead(msgIDs)
		assert.ThatError(t, err).IsNil()

		updatedMsgs := make([]Message, 0, n)

		wg := sync.WaitGroup{}

		mChan := make(chan Message)
		errChan := make(chan error)
		go folder.Messages(1, uint32(n), mChan, errChan)

		wg.Add(1)
		go func() {
			for {
				select {
				case msg := <-mChan:
					updatedMsgs = append(updatedMsgs, msg)
				case err := <-errChan:
					wg.Done()
					if err != nil {
						panic(err)
					}
				}
			}
		}()

		wg.Wait()

		assert.ThatSlice(t, updatedMsgs).HasLength(n)
		for _, m := range updatedMsgs {
			assert.That(t, m.Read).Equals(true)
		}
	})

	t.Run("delete messages", func(t *testing.T) {
		assert.That(t, len(msgIDs)).NotEqual(0)
		err = folder.Delete(msgIDs)
		assert.ThatError(t, err).IsNil()

		deletedMsgs := make([]Message, 0, n)
		wg := sync.WaitGroup{}
		mChan := make(chan Message)
		errChan := make(chan error)
		go folder.Messages(1, uint32(n), mChan, errChan)

		wg.Add(1)
		go func() {
			for {
				select {
				case msg := <-mChan:
					deletedMsgs = append(deletedMsgs, msg)
				case err := <-errChan:
					wg.Done()
					if err != nil {
						panic(err)
					}
				}
			}
		}()

		wg.Wait()

		assert.ThatSlice(t, deletedMsgs).HasLength(0)
	})

}
