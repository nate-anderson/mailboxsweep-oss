package mailsweeper

import (
	"fmt"
	"log"
	"net/mail"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/mcnijman/go-emailaddress"
)

type SMTPFolder struct {
	client  *client.Client
	mailbox *imap.MailboxStatus
}

func (f SMTPFolder) Count() uint32 {
	return f.mailbox.Messages
}

func (f SMTPFolder) Messages(start, end uint32, m chan<- Message, e chan<- error) {
	fmt.Println("messages called")
	seq := imap.SeqSet{}
	seq.AddRange(start, end)

	messageChan := make(chan *imap.Message)
	done := make(chan error)
	go func() {
		// set the components fetched here
		done <- f.client.Fetch(&seq, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}, messageChan)
	}()

	for {
		select {
		case msg := <-messageChan:
			if msg != nil {
				message, err := imapToMessage(*msg)
				if err != nil {
					close(m)
					e <- err
				}
				m <- message
				log.Println("Sent message on stream")
			}
		case err := <-done:
			close(m)
			e <- err
		}
	}
}

func imapToMessage(i imap.Message) (Message, error) {
	var sender *imap.Address
	if len(i.Envelope.Sender) >= 1 {
		sender = i.Envelope.Sender[0]
	} else if len(i.Envelope.From) >= 1 {
		sender = i.Envelope.From[0]
	} else {
		return Message{}, fmt.Errorf("message %s doesn't have a SENDER or FROM address", i.Envelope.MessageId)
	}
	var imapSenderName string
	senderEmail := sender.Address()
	parsed, err := mail.ParseAddress(senderEmail)
	if err == nil && parsed != nil {
		imapSenderName = parsed.Name
	}
	var username, domain string
	mailAddr, err := emailaddress.Parse(senderEmail)
	if err == nil && mailAddr != nil {
		username = mailAddr.LocalPart
		domain = mailAddr.Domain
	}
	return Message{
		Subject:   i.Envelope.Subject,
		ID:        i.Envelope.MessageId,
		InReplyTo: i.Envelope.InReplyTo,
		Sender: Sender{
			Name: firstNotEqual(
				"",
				sender.PersonalName,
				sender.MailboxName,
				imapSenderName,
				username,
				sender.Address(),
			),
			Address:  sender.Address(),
			Domain:   domain,
			Username: username,
		},
		SendDate: i.Envelope.Date,
	}, nil
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
