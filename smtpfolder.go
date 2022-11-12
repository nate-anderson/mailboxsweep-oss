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

func (f SMTPFolder) Messages(start, end uint32) ([]Message, error) {
	seq := imap.SeqSet{}
	seq.AddRange(start, end)

	messageChan := make(chan *imap.Message)
	done := make(chan error)
	go func() {
		// set the components fetched here
		done <- f.client.Fetch(&seq, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}, messageChan)
	}()

	messages := []Message{}
	for {
		select {
		case msg := <-messageChan:
			if msg != nil {
				messages = append(messages, imapToMessage(*msg))
			}
		case err := <-done:
			return messages, err
		}
	}
}

func imapToMessage(i imap.Message) Message {
	sender := i.Envelope.Sender[0]
	return Message{
		Subject: i.Envelope.Subject,
		ID:      i.Uid,
		Sender: Sender{
			Name: firstNotEqual(
				"",
				sender.PersonalName,
				sender.MailboxName,
				sender.Address(),
			),
			Address: sender.Address(),
		},
	}
}

// returns the first value in list not equal to not
// if all values equal not, it returns not
func firstNotEqual[T comparable](not T, list ...T) T {
	for _, item := range list {
		if item != not {
			return item
		}
	}
	return not
}
