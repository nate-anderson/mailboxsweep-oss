package mailsweeper

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type SMTPFolder struct {
	client  *client.Client
	mailbox *imap.MailboxStatus
}

func (f SMTPFolder) Count() uint32 {
	return f.mailbox.Messages
}

func (f SMTPFolder) Messages(start, end uint32) ([]imap.Message, error) {
	seq := imap.SeqSet{}
	seq.AddRange(start, end)

	messageChan := make(chan *imap.Message)
	done := make(chan error)
	go func() {
		// set the components fetched here
		done <- f.client.Fetch(&seq, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}, messageChan)
	}()

	messages := []imap.Message{}
	for {
		select {
		case msg := <-messageChan:
			if msg != nil {
				messages = append(messages, *msg)
			}
		case err := <-done:
			return messages, err
		}
	}
}
