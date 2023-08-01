package inbox

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/metering"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/sirupsen/logrus"
)

func TestIMAPFolder(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

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
			t.Errorf("failed to send test email with testConf %+v: %s", testConf, err)
		}
	}

	usageRepo := metering.NewUsageRepository(db, table)

	provider, err := NewIMAPProvider(
		ctx,
		testConf.IMAPHost,
		testConf.IMAPPort,
		testConf.Address,
		usageRepo,
		nil,
		logrus.New(),
	)
	assert.ThatError(t, err).IsNil()

	imapCred := inboxcredential.NewIMAPInboxCredential(testConf.Address, testConf.Password)

	inbox, err := provider.Login(
		ctx,
		testConf.Address,
		imapCred,
	)
	assert.ThatError(t, err).IsNil()

	folder, err := inbox.Folder("INBOX")
	assert.ThatError(t, err).IsNil()

	msgs := make([]Message, 0, n)

	t.Run("get messages", func(t *testing.T) {
		mChan := make(chan Message)
		errChan := make(chan error)

		wg := sync.WaitGroup{}

		go folder.Messages(ctx, 1, uint32(n), mChan, errChan)

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
		msgIDs = append(msgIDs, m.UID)
	}

	t.Run("mark messages read", func(t *testing.T) {
		assert.That(t, len(msgIDs)).NotEqual(0)
		err := folder.MarkRead(ctx, msgIDs)
		assert.ThatError(t, err).IsNil()

		updatedMsgs := make([]Message, 0, n)

		wg := sync.WaitGroup{}

		mChan := make(chan Message)
		errChan := make(chan error)
		go folder.Messages(ctx, 1, uint32(n), mChan, errChan)

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
		err = folder.Delete(ctx, msgIDs)
		assert.ThatError(t, err).IsNil()

		deletedMsgs := make([]Message, 0, n)
		wg := sync.WaitGroup{}
		mChan := make(chan Message)
		errChan := make(chan error)
		go folder.Messages(ctx, 1, uint32(n), mChan, errChan)

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

func TestIMAPParse(t *testing.T) {
	m := imap.Message{
		SeqNum: 417,
		Uid:    1625,
		Flags:  []string{"\\Seen"},
		Envelope: &imap.Envelope{
			Date:    time.Now(),
			Subject: "Your 1st Survey from NPDOR: Music",
			Sender: []*imap.Address{
				{
					PersonalName: "",
					AtDomainList: "",
					MailboxName:  "<Survey",
					HostName:     "npdor.com>",
				},
			},
			InReplyTo: "<AcnEix7kPhoeqt4QRtKQTxvefbq+6A==>",
			MessageId: "<49f12f28.1467f10a.1351.ffffbcfcSMTPIN_ADDED@mx.google.com>",
		},
	}

	parsed, err := imapToMessage(m, "INBOX")
	assert.ThatError(t, err).IsNil()
	assert.That(t, parsed.Sender.Address).Equals("Survey@npdor.com")
}
